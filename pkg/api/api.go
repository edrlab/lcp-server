// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// Package api manages api handlers (controllers)
package api

import (
	"crypto/tls"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
)

// HandleCtx contains the context required by handlers.
type HandlerCtx struct {
	*conf.Config
	stor.Store
	Cert *tls.Certificate
}

// NewHandlerCtx returns a new handler context
func NewHandlerCtx(cf *conf.Config, st stor.Store, cr *tls.Certificate) *HandlerCtx {
	return &HandlerCtx{
		Config: cf,
		Store:  st,
		Cert:   cr,
	}
}
