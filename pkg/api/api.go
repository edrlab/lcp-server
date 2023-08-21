// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// Package api manages the api controllers
package api

import (
	"crypto/tls"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
)

// APICtrl contains the context required by http handlers.
type APICtrl struct {
	*conf.Config // TODO: change for an interface (dependency)
	stor.Store
	Cert *tls.Certificate
}

// NewAPICtrl returns a new API controller
func NewAPICtrl(cf *conf.Config, st stor.Store, cr *tls.Certificate) *APICtrl {
	return &APICtrl{
		Config: cf,
		Store:  st,
		Cert:   cr,
	}
}
