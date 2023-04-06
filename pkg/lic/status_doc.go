// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"errors"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
)

// StatusDoc data model
type (
	StatusDoc struct {
		ID              string           `json:"id"`
		Status          string           `json:"status"`
		Message         string           `json:"message"`
		Updated         Updated          `json:"updated"`
		Links           []Link           `json:"links"`
		PotentialRights *PotentialRights `json:"potential_rights,omitempty"`
		Events          []stor.Event     `json:"events,omitempty"`
	}

	Updated struct {
		License *time.Time `json:"license"`
		Status  *time.Time `json:"status"`
	}

	PotentialRights struct {
		End *time.Time `json:"end,omitempty"`
	}

	// License management interface
	LicenseManager interface {
		Register(license *stor.LicenseInfo) error
		Renew(license *stor.LicenseInfo) error
		Return(license *stor.LicenseInfo) error
		Revoke(license *stor.LicenseInfo) error
	}

	LicenseHandler struct {
		*conf.Config // TODO: change for an interface (dependency)
		stor.Store
	}
)

func NewLicenseHandler(cf *conf.Config, st stor.Store) *LicenseHandler {
	return &LicenseHandler{
		Config: cf,
		Store:  st,
	}
}

// ====

// NewStatusDoc returns a Status Document
func NewStatusDoc(license *stor.LicenseInfo) *StatusDoc {
	statusDoc := StatusDoc{
		ID:      license.UUID,
		Status:  license.Status,
		Message: "License status", // TODO: flexible, localize
		Updated: Updated{
			License: license.Updated,
			Status:  license.StatusUpdated,
		},
	}
	return &statusDoc
}

// Register records a new device using a license
func (lh *LicenseHandler) Register(licenseID string) (*stor.LicenseInfo, error) {

	// Get license info
	licInfo, err := lh.Store.License().Get(licenseID)
	if err != nil {
		return nil, errors.New("failed to get license info")
	}

	// check that the license is in ready or active status
	if (licInfo.Status != stor.STATUS_ACTIVE) && (licInfo.Status != stor.STATUS_READY) {
		return nil, errors.New("registering a device on an inactive license is not allowed")
	}

	// check that the device has not already been registered for this license

	return licInfo, nil
}
