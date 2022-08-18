// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// Package api manages api handlers (controllers)
package api

import (
	"github.com/edrlab/lcp-server/pkg/stor"
)

// HandleCtx contains the context required by handlers.
type HandlerCtx struct {
	St stor.Store
}

// NewBaseHandler returns a new HandlerCtx
func NewHandlerCtx(st stor.Store) *HandlerCtx {
	return &HandlerCtx{
		St: st,
	}
}
