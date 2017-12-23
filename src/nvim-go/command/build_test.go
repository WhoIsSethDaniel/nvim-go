package command

import (
	"context"
	"path/filepath"
	"testing"

	buildctx "nvim-go/ctx"
	"nvim-go/nvimutil"
	"nvim-go/testutil"

	"github.com/neovim/go-client/nvim"
	"golang.org/x/sync/syncmap"
)

func TestCommand_Build(t *testing.T) {
	ctx := testutil.TestContext(context.Background())

	type fields struct {
		ctx       context.Context
		Nvim      *nvim.Nvim
		buildctxt *buildctx.Context
		errs      *syncmap.Map
	}
	type args struct {
		bang bool
		eval *CmdBuildEval
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		// want interface{}
		wantErr bool
	}{
		{
			name: "nvim-go File: filepath.Join(projectRoot, \"src/nvim-go/command\")",
			fields: fields{
				ctx:       ctx,
				Nvim:      nvimutil.TestNvim(t, "testdata"),
				buildctxt: buildctx.NewContext(),
			},
			args: args{
				eval: &CmdBuildEval{
					Cwd:  testdataPath,
					File: testdataPath,
				},
			},
			wantErr: false,
		},
		{
			name: "gsftp",
			fields: fields{
				ctx:       ctx,
				Nvim:      nvimutil.TestNvim(t, gsftpRoot),
				buildctxt: buildctx.NewContext(),
			},
			args: args{
				eval: &CmdBuildEval{
					Cwd:  gsftpRoot,
					File: gsftpRoot,
				},
			},
			wantErr: false,
		},
		{
			name: "correct (astdump)",
			fields: fields{
				ctx:       ctx,
				Nvim:      nvimutil.TestNvim(t, filepath.Join(astdump, "astdump.go")), // correct file
				buildctxt: buildctx.NewContext(),
			},
			args: args{
				eval: &CmdBuildEval{
					Cwd:  astdump,
					File: filepath.Join(astdump, "astdump.go"),
				},
			},
			wantErr: false,
		},
		{
			name: "broken (broken)",
			fields: fields{
				ctx:       ctx,
				Nvim:      nvimutil.TestNvim(t, brokenMain), // broken file
				buildctxt: buildctx.NewContext(),
			},
			args: args{
				eval: &CmdBuildEval{
					Cwd:  broken,
					File: brokenMain,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewCommand(tt.fields.ctx, tt.fields.Nvim, tt.fields.buildctxt)
			c.buildctxt.SetContext(filepath.Dir(tt.args.eval.File))

			err := c.Build(tt.args.bang, tt.args.eval)
			if e, ok := err.(error); ok {
				if (err != nil) != tt.wantErr {
					t.Errorf("err: %v, wantErr %v", e, tt.wantErr)
					return
				}
			}
			if errlist, ok := err.([]*nvim.QuickfixError); ok {
				if (len(errlist) != 0) != tt.wantErr {
					t.Errorf("%q. Commands.Build(%v, %v)", tt.name, tt.args.bang, tt.args.eval)
					return
				}
			}
		})
	}
}

func BenchmarkBuildGo(b *testing.B) {
	ctx := testutil.TestContext(context.Background())
	buildctxt := buildctx.NewContext()
	c := NewCommand(ctx, benchVim(b, astdumpMain), buildctxt)

	for i := 0; i < b.N; i++ {
		c.Build(false, &CmdBuildEval{
			Cwd:  astdump,
			File: astdump,
		})
		if len(c.buildctxt.Errlist) != 0 {
			b.Errorf("BenchmarkBuildGo: %v", c.buildctxt.Errlist)
		}
	}
}

func BenchmarkBuildGb(b *testing.B) {
	ctx := testutil.TestContext(context.Background())
	buildctxt := buildctx.NewContext()
	c := NewCommand(ctx, benchVim(b, gsftpMain), buildctxt)

	for i := 0; i < b.N; i++ {
		c.Build(false, &CmdBuildEval{
			Cwd:  gsftpRoot,
			File: gsftpRoot,
		})
		if len(c.buildctxt.Errlist) != 0 {
			b.Errorf("BenchmarkBuildGb: %v", c.buildctxt.Errlist)
		}
	}
}

// func TestCommand_compileCmd(t *testing.T) {
// 	gobinary, err := exec.LookPath("go")
// 	if err != nil {
// 		t.Error(err)
// 	}
// 	gbbinary, err := exec.LookPath("gb")
// 	if err != nil {
// 		t.Error(err)
// 	}
//
// 	type fields struct {
// 		Nvim  *nvim.Nvim
// 		ctx  *ctx.Context
// 	}
// 	type args struct {
// 		bang bool
// 		dir  string
// 	}
// 	tests := []struct {
// 		name     string
// 		fields   fields
// 		args     args
// 		want     string
// 		wantErr  bool
// 		testfile bool
// 	}{
// 		{
// 			name: "astdump (go build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: astdump,
// 			},
// 			want:    gobinary,
// 			wantErr: false,
// 		},
// 		{
// 			name: "nvim-go (gb build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: "testdata",
// 			},
// 			want:    gbbinary,
// 			wantErr: false,
// 		},
// 		{
// 			name: "gsftp (gb build)",
// 			fields: fields{
// 				Nvim: nvimutil.TestNvim(t, "testdata"),
// 				ctx: ctx.NewContext(),
// 			},
// 			args: args{
// 				dir: gsftpRoot,
// 			},
// 			want:    gbbinary,
// 			wantErr: false,
// 		},
// 	}
// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			c := NewCommand(tt.fields.Nvim, tt.fields.ctx)
// 			c.ctx.SetContext(filepath.Dir(tt.args.dir))
//
// 			got, err := c.compileCmd(tt.args.bang, tt.args.dir, tt.testfile)
// 			if (err != nil) != tt.wantErr {
// 				t.Errorf("Commands.compileCmd(%v, %v) error = %v, wantErr %v", tt.args.bang, tt.args.dir, err, tt.wantErr)
// 				return
// 			}
// 			if !reflect.DeepEqual(got.Args[0], tt.want) {
// 				t.Errorf("Commands.compileCmd(%v, %v) = %v, want %v", tt.args.bang, tt.args.dir, got, tt.want)
// 			}
// 		})
// 	}
// }
