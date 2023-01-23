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

// APIHandler contains the context required by http handlers.
type APIHandler struct {
	*conf.Config // TODO: change for an interface (dependency)
	stor.Store
	Cert *tls.Certificate
}

// NewAPIHandler returns a new API context
func NewAPIHandler(cf *conf.Config, st stor.Store, cr *tls.Certificate) *APIHandler {
	return &APIHandler{
		Config: cf,
		Store:  st,
		Cert:   cr,
	}
}
