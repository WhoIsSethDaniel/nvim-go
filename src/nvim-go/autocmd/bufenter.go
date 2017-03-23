// Copyright 2017 The nvim-go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package autocmd

type BufEnterEval struct {
	BufNr int    `eval:"bufnr('%')"`
	WinID int    `eval:"win_getid()"`
	Dir   string `eval:"expand('%:p:h')"`
}

func (a *Autocmd) BufEnter(eval *BufEnterEval) {
	a.mu.Lock()
	a.ctx.BufNr = eval.BufNr
	a.ctx.WinID = eval.WinID
	a.mu.Unlock()

	a.ctx.SetContext(eval.Dir)
}