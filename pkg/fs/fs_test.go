// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package fs_test

import (
	"go/build"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/neovim/go-client/nvim"

	"github.com/zchee/nvim-go/pkg/fs"
	"github.com/zchee/nvim-go/pkg/internal/testutil"
	"github.com/zchee/nvim-go/pkg/nvimutil"
)

func TestChdir(t *testing.T) {
	testCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		v   *nvim.Nvim
		dir string
	}
	tests := []struct {
		name    string
		args    args
		wantCwd string
	}{
		{
			name: "gb/gsftp",
			args: args{
				v:   nvimutil.TestNvim(t, testGbRoot),
				dir: filepath.Join(testGbRoot, "src", "gsftp"),
			},
			wantCwd: filepath.Join(testGbRoot, "src", "gsftp"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if testCwd == tt.args.dir {
					t.Errorf("%q. Chdir(%v, %v) = %v, want %v", tt.name, tt.args.v, tt.args.dir, testCwd, tt.wantCwd)
				}
			}()
			defer fs.Chdir(tt.args.v, tt.args.dir)()
			var cwd interface{}
			tt.args.v.Eval("getcwd()", &cwd)
			if cwd.(string) != testCwd {
				t.Errorf("%q. Chdir(%v, %v) = %v, want %v", tt.name, tt.args.v, tt.args.dir, cwd, tt.wantCwd)
			}
		})
	}
}

func TestTrimGoPath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "typical go package full path",
			args: args{p: filepath.Join(testGoPath, "src/github.com/zchee/nvim-go")},
			want: "github.com/zchee/nvim-go",
		},
		{
			name: ".goimportsignore file on GOPATH",
			args: args{p: filepath.Join(testGoPath, ".goimportsignore")},
			want: ".goimportsignore",
		},
		{
			name: ".a file on GOPATH",
			args: args{p: filepath.Join(testGoPath, ".a")},
			want: ".a",
		},
		{
			name: "just GOPATH only",
			args: args{p: testGoPath},
			want: "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer testutil.SetBuildContext(t, testGoPath)()
			if got := fs.TrimGoPath(tt.args.p); got != tt.want {
				t.Errorf("TrimGoPath(%v) = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}

func TestJoinGoPath(t *testing.T) {
	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "astdump",
			args: args{p: "astdump"},
			want: filepath.Join(testGoPath, "src", "astdump"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			defer testutil.SetBuildContext(t, testGoPath)()
			if got := fs.JoinGoPath(tt.args.p); got != tt.want {
				t.Errorf("JoinGoPath(%v) = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}

func TestShortFilePath(t *testing.T) {
	testCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		p   string
		cwd string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "filename only",
			args: args{
				p:   filepath.Join(testCwd, "nvim-go/fs/fs_test.go"),
				cwd: filepath.Join(testCwd, "nvim-go/fs"),
			},
			want: "./fs_test.go",
		},
		{
			name: "with directory",
			args: args{
				p:   filepath.Join(testCwd, "nvim-go/fs/fs_test.go"),
				cwd: filepath.Join(testCwd, "nvim-go"),
			},
			want: "./fs/fs_test.go",
		},
		{
			name: "not shorten",
			args: args{
				p:   filepath.Join(testCwd, "nvim-go/fs/fs_test.go"),
				cwd: filepath.Join(testCwd, "nvim-go/command"),
			},
			want: filepath.Join(testCwd, "nvim-go/fs/fs_test.go"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.ShortFilePath(tt.args.p, tt.args.cwd); got != tt.want {
				t.Errorf("ShortFilePath(%v, %v) = %v, want %v", tt.args.p, tt.args.cwd, got, tt.want)
			}
		})
	}
}

func TestRel(t *testing.T) {
	testCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		f   string
		cwd string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "own filepath and directory",
			args: args{
				f:   filepath.Join(testCwd, "fs_test.go"),
				cwd: testCwd,
			},
			want: "fs_test.go",
		},
		{
			name: "own filepath and project root",
			args: args{
				f:   filepath.Join(testCwd, "fs_test.go"),
				cwd: filepath.Join(build.Default.GOPATH, "src", "github.com", "zchee", "nvim-go"),
			},
			want: "pkg/fs/fs_test.go",
		},
		{
			name: "Use different directory",
			args: args{
				f:   filepath.Join(testCwd, "fs_test.go"),
				cwd: filepath.Join(testCwd, "../commands"),
			},
			want: "../fs/fs_test.go",
		},
		{
			name: "Fail the filepath.Rel()",
			args: args{
				f:   filepath.Join(testCwd, "fs_test.go"),
				cwd: filepath.Join("foo", "bar", "baz"),
			},
			want: filepath.Join(testCwd, "fs_test.go"),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.Rel(tt.args.cwd, tt.args.f); got != tt.want {
				t.Errorf("Rel(%v, %v) = got %v, want %v", tt.args.f, tt.args.cwd, got, tt.want)
			}
		})
	}
}

func TestExpandGoRoot(t *testing.T) {
	goroot := runtime.GOROOT()

	type args struct {
		p string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "exist $GOROOT",
			args: args{p: "$GOROOT/src/go/ast/ast.go"},
			want: filepath.Join(goroot, "src/go/ast/ast.go"),
		},
		{
			name: "not exist $GOROOT",
			args: args{p: "src/go/ast/ast.go"},
			want: "src/go/ast/ast.go",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.ExpandGoRoot(tt.args.p); got != tt.want {
				t.Errorf("ExpandGoRoot(%v) = %v, want %v", tt.args.p, got, tt.want)
			}
		})
	}
}

func TestIsDir(t *testing.T) {
	testCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "true (own parent directory)",
			args: args{filename: testCwd},
			want: true,
		},
		{
			name: "false (own file path)",
			args: args{filename: filepath.Join(testCwd, "fs_test.go")},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.IsDir(tt.args.filename); got != tt.want {
				t.Errorf("IsDir(%v) = %v, want %v", tt.args.filename, got, tt.want)
			}
		})
	}
}

func TestIsExist(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "exist (own file)",
			args: args{filename: "./fs_test.go"},
			want: true,
		},
		{
			name: "not exist",
			args: args{filename: "./not_exist.go"},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.IsExist(tt.args.filename); got != tt.want {
				t.Errorf("IsExist(%v) = %v, want %v", tt.args.filename, got, tt.want)
			}
		})
	}
}

func TestIsNotExist(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "exist (own file)",
			args: args{filename: "./fs_test.go"},
			want: false,
		},
		{
			name: "not exist",
			args: args{filename: "./not_exist.go"},
			want: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.IsNotExist(tt.args.filename); got != tt.want {
				t.Errorf("IsExist(%v) = %v, want %v", tt.args.filename, got, tt.want)
			}
		})
	}
}

func TestIsGoFile(t *testing.T) {
	type args struct {
		filename string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "go file",
			args: args{filename: "fs.go"},
			want: true,
		},
		{
			name: "not go file",
			args: args{filename: "test.c"},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fs.IsGoFile(tt.args.filename); got != tt.want {
				t.Errorf("IsGoFile(%v) = %v, want %v", tt.args.filename, got, tt.want)
			}
		})
	}
}
