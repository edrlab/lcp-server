// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"testing"

	"github.com/edrlab/lcp-server/pkg/stor"
)

func TestRegister(t *testing.T) {

	// use the globally defined Licinfo
	licInfo, err := LicHandler.Register(LicInfo.UUID)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to register a license.")
	}

	if licInfo.Status != stor.STATUS_ACTIVE {
		t.Errorf("expecter status active, got %s", licInfo.Status)
	}

}
