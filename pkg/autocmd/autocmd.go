// Copyright 2016 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocmd

import (
	"context"
	"sync"

	"github.com/neovim/go-client/nvim"
	"go.opencensus.io/trace"

	"github.com/zchee/nvim-go/pkg/buildctxt"
	"github.com/zchee/nvim-go/pkg/command"
)

// Autocmd represents a autocmd context.
type Autocmd struct {
	Nvim         *nvim.Nvim
	buildContext *buildctxt.Context
	cmd          *command.Command
	mu           sync.Mutex

	bufWritePostChan chan error
	bufWritePreChan  chan interface{}
	wg               sync.WaitGroup

	errs *sync.Map
}

func (a *Autocmd) getStatus(ctx context.Context, bufnr, winID int, dir string) {
	span := trace.FromContext(ctx)
	span.SetName("getStatus")
	defer span.End()

	a.mu.Lock()
	a.buildContext.BufNr = bufnr
	a.buildContext.WinID = winID
	a.buildContext.Dir = dir
	a.mu.Unlock()
}
