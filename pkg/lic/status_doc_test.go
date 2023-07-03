// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"testing"

	"github.com/edrlab/lcp-server/pkg/stor"
)

func TestRegister(t *testing.T) {

	deviceInfo := &DeviceInfo{
		ID:   "1",
		Name: "device1",
	}

	// use the globally defined LicHandler and Licinfo
	statusDoc, err := LicHandler.Register(LicInfo.UUID, deviceInfo)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to register a license.")
	}

	if statusDoc.Status != stor.STATUS_ACTIVE {
		t.Errorf("expected an active status, got %s", statusDoc.Status)
	}

}

func TestRenew(t *testing.T) {

	deviceInfo := &DeviceInfo{
		ID:   "1",
		Name: "device1",
	}

	statusDoc, err := LicHandler.Register(LicInfo.UUID, deviceInfo)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to register a license.")
	}

	if statusDoc.Status != stor.STATUS_ACTIVE {
		t.Errorf("expected an active status, got %s", statusDoc.Status)
	}

	statusDoc, err = LicHandler.Renew(LicInfo.UUID, deviceInfo, nil)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to renew a license.")
	}

	if statusDoc.Status != stor.STATUS_ACTIVE {
		t.Errorf("expected an active status, got %s", statusDoc.Status)
	}

}

func TestRevoke(t *testing.T) {

	deviceInfo := &DeviceInfo{
		ID:   "1",
		Name: "device1",
	}

	statusDoc, err := LicHandler.Register(LicInfo.UUID, deviceInfo)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to register a license.")
	}

	if statusDoc.Status != stor.STATUS_ACTIVE {
		t.Errorf("expected an active status, got %s", statusDoc.Status)
	}

	statusDoc, err = LicHandler.Revoke(LicInfo.UUID)
	if err != nil {
		t.Log(err)
		t.Fatal("failed to revoke a license.")
	}

	if statusDoc.Status != stor.STATUS_REVOKED {
		t.Errorf("expected a revoked status, got %s", statusDoc.Status)
	}

}
