// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"context"
	"flag"
	logpkg "log"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"syscall"

	"cloud.google.com/go/profiler"
	"contrib.go.opencensus.io/exporter/stackdriver"
	"github.com/neovim/go-client/nvim/plugin"
	"github.com/pkg/errors"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/trace"
	"go.uber.org/multierr"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"golang.org/x/exp/errors/fmt"
	"golang.org/x/sync/errgroup"

	"github.com/zchee/nvim-go/pkg/autocmd"
	"github.com/zchee/nvim-go/pkg/buildctxt"
	"github.com/zchee/nvim-go/pkg/command"
	"github.com/zchee/nvim-go/pkg/config"
	"github.com/zchee/nvim-go/pkg/logger"
	"github.com/zchee/nvim-go/pkg/nctx"
	"github.com/zchee/nvim-go/pkg/server"
	"github.com/zchee/nvim-go/pkg/version"
)

// flags
var (
	fVersion    = flag.Bool("version", false, "Show the version information.")
	pluginHost  = flag.String("manifest", "", "Write plugin manifest for `host` to stdout")
	vimFilePath = flag.String("location", "", "Manifest is automatically written to `.vim file`")
)

func init() {
	flag.Parse()
	logpkg.SetPrefix("nvim-go: ")
}

func main() {
	if *fVersion {
		fmt.Printf("%s:\n  version: %s\n", nctx.AppName, version.Version)
		return
	}

	ctx, cancel := context.WithCancel(Context())
	defer cancel()

	if *pluginHost != "" {
		os.Unsetenv("NVIM_GO_DEBUG")               // disable zap output
		ctx = logger.NewContext(ctx, zap.NewNop()) // avoid nil panic on logger.FromContext

		fn := func(p *plugin.Plugin) error {
			return func(ctx context.Context, p *plugin.Plugin) error {
				bctxt := buildctxt.NewContext()
				c := command.Register(ctx, p, bctxt)
				autocmd.Register(ctx, p, bctxt, c)
				return nil
			}(ctx, p)
		}
		if err := Plugin(fn); err != nil {
			logpkg.Fatal(err)
		}
		return
	}

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	sighupFn := func() {}
	sigintFn := func() {
		logpkg.Println("Start shutdown gracefully")
		cancel()
	}
	go signalHandler(sigc, sighupFn, sigintFn)

	errc := make(chan error, 1)
	go func() {
		defer close(errc)
		errc <- startServer(ctx)
	}()

	select {
	case <-ctx.Done():
	case err := <-errc:
		if err != nil {
			logpkg.Fatal(err)
		}
	}
	logpkg.Println("shutdown nvim-go server")
}

func signalHandler(ch <-chan os.Signal, sighupFn, sigintFn func()) {
	for {
		select {
		case sig := <-ch:
			switch sig {
			case syscall.SIGHUP:
				logpkg.Printf("catch signal %s", sig)
				sighupFn()
			case syscall.SIGINT, syscall.SIGTERM:
				logpkg.Printf("catch signal %s", sig)
				sigintFn()
			}
		}
	}
}

func startServer(ctx context.Context) (errs error) {
	env := config.Process()

	var lv zapcore.Level
	if err := lv.UnmarshalText([]byte(env.LogLevel)); err != nil {
		return fmt.Errorf("failed to parse log level: %s, err: %v", env.LogLevel, err)
	}
	log, undo := logger.NewRedirectZapLogger(lv)
	defer undo()
	ctx = logger.NewContext(ctx, log)

	if gcpProjectID, ok := config.HasGCPProjectID(); ok {
		// OpenCensus tracing with Stackdriver exporter
		sdOpts := stackdriver.Options{
			ProjectID: gcpProjectID,
			OnError: func(err error) {
				errs = multierr.Append(errs, fmt.Errorf("stackdriver.Exporter: %v", err))
			},
			MetricPrefix: nctx.AppName,
			Context:      ctx,
		}
		sd, err := stackdriver.NewExporter(sdOpts)
		if err != nil {
			logpkg.Fatalf("failed to create stackdriver exporter: %v", err)
		}
		defer sd.Flush()
		trace.RegisterExporter(sd)
		view.RegisterExporter(sd)
		log.Info("opencensus", zap.String("trace", "enabled Stackdriver exporter"))

		// Stackdriver Profiler
		profConf := profiler.Config{
			Service:        nctx.AppName,
			ServiceVersion: version.Tag,
			MutexProfiling: true,
			ProjectID:      gcpProjectID,
		}
		if err := profiler.Start(profConf); err != nil {
			logpkg.Fatalf("failed to start stackdriver profiler: %v", err)
		}
		log.Info("stackdriver", zap.String("profiler", "enabled Stackdriver profiler"))

		trace.ApplyConfig(trace.Config{DefaultSampler: trace.AlwaysSample()})
		var span *trace.Span
		ctx, span = trace.StartSpan(ctx, "main") // start root span
		defer span.End()
	}

	fn := func(p *plugin.Plugin) error {
		return func(ctx context.Context, p *plugin.Plugin) error {
			log := logger.FromContext(ctx).Named("main")
			ctx = logger.NewContext(ctx, log)

			bctxt := buildctxt.NewContext()
			cmd := command.Register(ctx, p, bctxt)
			autocmd.Register(ctx, p, bctxt, cmd)

			// switch to unix socket rpc-connection
			if n, err := server.Dial(ctx); err == nil {
				p.Nvim = n
			}

			return nil
		}(ctx, p)
	}

	eg := new(errgroup.Group)
	eg.Go(func() error {
		return Plugin(fn)
	})
	eg.Go(func() error {
		return subscribeServer(ctx)
	})

	log.Info(fmt.Sprintf("starting %s server", nctx.AppName), zap.Object("env", env))
	if err := eg.Wait(); err != nil {
		log.Fatal("occurred error", zap.Error(err))
	}

	return errs
}

func subscribeServer(ctx context.Context) error {
	log := logger.FromContext(ctx).Named("child")
	ctx = logger.NewContext(ctx, log)

	s, err := server.NewServer(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to create NewServer")
	}
	go s.Serve()

	s.Nvim.Subscribe(nctx.Method)

	select {
	case <-ctx.Done():
		if err := s.Close(); err != nil {
			log.Fatal("Close", zap.Error(err))
		}
		return nil
	}
}
