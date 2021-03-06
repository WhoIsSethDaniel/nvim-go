// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package command

import (
	"context"
	"encoding/json"
	"os/exec"
	"sort"
	"strings"

	"github.com/neovim/go-client/nvim"
	"github.com/pkg/errors"
	"go.opencensus.io/trace"

	"github.com/zchee/nvim-go/pkg/config"
	"github.com/zchee/nvim-go/pkg/fs"
	"github.com/zchee/nvim-go/pkg/monitoring"
	"github.com/zchee/nvim-go/pkg/nvimutil"
)

func (c *Command) cmdMetalinter(ctx context.Context, cwd string) {
	errch := make(chan interface{}, 1)
	go func() {
		errch <- c.Metalinter(ctx, cwd)
	}()

	select {
	case <-ctx.Done():
		return
	case err := <-errch:
		switch e := err.(type) {
		case error:
			nvimutil.ErrorWrap(c.Nvim, e)
		case nil:
			// nothing to do
		}
	}
}

type metalinterResult struct {
	Linter   string `json:"linter"`   // name of linter tool
	Severity string `json:"severity"` // result of type
	Path     string `json:"path"`     // path of file
	Line     int    `json:"line"`     // line of file
	Col      int    `json:"col"`      // col of file
	Message  string `json:"message"`  // description of linter message
}

// Metalinter lint the Go sources from current buffer's package use gometalinter tool.
func (c *Command) Metalinter(ctx context.Context, cwd string) error {
	var span *trace.Span
	ctx, span = monitoring.StartSpan(ctx, "MetaLinter")
	defer span.End()

	var loclist []*nvim.QuickfixError
	w := nvim.Window(c.buildContext.WinID)

	var args []string
	switch c.buildContext.Build.Tool {
	case "go":
		args = append(args, cwd+"/...")
	case "gb":
		args = append(args, c.buildContext.Build.ProjectRoot+"/...")
	}
	args = append(args, []string{"--json", "--disable-all", "--deadline", config.MetalinterDeadline}...)

	for _, t := range config.MetalinterTools {
		args = append(args, "--enable", t)
	}
	if len(config.MetalinterSkipDir) != 0 {
		for _, dir := range config.MetalinterSkipDir {
			args = append(args, "--skip", dir)
		}
	}

	cmd := exec.Command("gometalinter", args...)
	stdout, err := cmd.Output()
	cmd.Run()

	var result = []metalinterResult{}
	if err != nil {
		if err := json.Unmarshal(stdout, &result); err != nil {
			span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: err.Error()})
			return errors.WithStack(err)
		}
	}

	sort.Sort(byPath(result))

	for _, r := range result {
		loclist = append(loclist, &nvim.QuickfixError{
			FileName: fs.Rel(r.Path, cwd),
			LNum:     r.Line,
			Col:      r.Col,
			Text:     r.Linter + ": " + r.Message,
			Type:     strings.ToUpper(r.Severity[:1]),
		})
	}

	if err := nvimutil.SetLoclist(c.Nvim, loclist); err != nil {
		span.SetStatus(trace.Status{Code: trace.StatusCodeInternal, Message: err.Error()})
		return nvimutil.ErrorWrap(c.Nvim, errors.WithStack(err))
	}
	return nvimutil.OpenLoclist(c.Nvim, w, loclist, true)
}

type byPath []metalinterResult

func (a byPath) Len() int           { return len(a) }
func (a byPath) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a byPath) Less(i, j int) bool { return a[i].Path < a[j].Path }
