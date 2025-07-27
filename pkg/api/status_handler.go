// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

var (
	ErrMissingLicenseId  = errors.New("missing required license identifier")
	ErrMissingDeviceInfo = errors.New("missing required device information")
)

// Status returns a status document for the input license.
func (a *APICtrl) StatusDoc(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var licenseID string
	if licenseID = getLicenseID(w, r); licenseID == "" {
		return
	}

	lh := lic.NewLicenseCtrl(a.Config, a.Store)

	// get license info
	license, err := a.Store.License().Get(licenseID)
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// generate a status document
	statusDoc := lh.NewStatusDoc(license)
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// Register records a new device using the license and returns a status document.
func (a *APICtrl) Register(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var licenseID string
	if licenseID = getLicenseID(w, r); licenseID == "" {
		return
	}
	var deviceInfo *lic.DeviceInfo
	if deviceInfo = getDeviceInfo(w, r); deviceInfo == nil {
		return
	}

	lh := lic.NewLicenseCtrl(a.Config, a.Store)

	// register
	statusDoc, err := lh.Register(licenseID, deviceInfo)
	if err != nil {
		render.Render(w, r, ErrRegister(err))
		return
	}
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// Renew extends the lifetime of a license and returns a status document.
func (a *APICtrl) Renew(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var licenseID string
	if licenseID = getLicenseID(w, r); licenseID == "" {
		return
	}
	var deviceInfo *lic.DeviceInfo
	if deviceInfo = getDeviceInfo(w, r); deviceInfo == nil {
		return
	}
	// check the presence of the new end date (optional)
	var newEnd *time.Time
	var err error
	if newEnd, err = getNewEnd(w, r); err != nil {
		return
	}

	lh := lic.NewLicenseCtrl(a.Config, a.Store)

	// renew
	statusDoc, err := lh.Renew(licenseID, deviceInfo, newEnd)
	if err != nil {
		render.Render(w, r, ErrRenew(err))
		return
	}
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// Return forces the expiration of a license and returns a status document.
func (a *APICtrl) Return(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var licenseID string
	if licenseID = getLicenseID(w, r); licenseID == "" {
		return
	}
	var deviceInfo *lic.DeviceInfo
	if deviceInfo = getDeviceInfo(w, r); deviceInfo == nil {
		return
	}

	lh := lic.NewLicenseCtrl(a.Config, a.Store)

	// return
	statusDoc, err := lh.Return(licenseID, deviceInfo)
	if err != nil {
		render.Render(w, r, ErrReturn(err))
		return
	}
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
	}

}

// Revoke forces the expiration of a license and returns a status document.
func (a *APICtrl) Revoke(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var licenseID string
	if licenseID = getLicenseID(w, r); licenseID == "" {
		return
	}

	lh := lic.NewLicenseCtrl(a.Config, a.Store)

	// revoke
	statusDoc, err := lh.Revoke(licenseID)
	if err != nil {
		render.Render(w, r, ErrRevoke(err))
		return
	}
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
	}

}

// --
// local functions
// --

func getLicenseID(w http.ResponseWriter, r *http.Request) (licenseID string) {

	if licenseID = chi.URLParam(r, "licenseID"); licenseID == "" {
		render.Render(w, r, ErrInvalidRequest(ErrMissingLicenseId))
	}
	return
}

// getDeviceInfo gets the device id and name from URL Query parameters
func getDeviceInfo(w http.ResponseWriter, r *http.Request) *lic.DeviceInfo {

	var device lic.DeviceInfo

	device.ID = r.URL.Query().Get("id")
	device.Name = r.URL.Query().Get("name")

	dILen := len(device.ID)
	dNLen := len(device.Name)

	if (dILen == 0) || (dNLen == 0) {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required device identifier and name")))
		return nil
	}
	if (dILen > 255) || (dNLen > 255) {
		render.Render(w, r, ErrInvalidRequest(errors.New("device identifier and name must be shorter")))
		return nil
	}
	return &device
}

func getNewEnd(w http.ResponseWriter, r *http.Request) (*time.Time, error) {

	endParam := r.URL.Query().Get("end")
	if endParam == "" {
		return nil, nil
	}
	newEnd, err := time.Parse(time.RFC3339, endParam)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(errors.New("invalid date end parameter")))
		return nil, err

	}
	return &newEnd, nil
}

// --
// Request and Response payloads for the REST api.
// --

// LicenseResponse is the response payload for licenses.
type StatusDocResponse struct {
	*lic.StatusDoc
}

// NewLicenseResponse creates a rendered license
func NewStatusDocResponse(statusDoc *lic.StatusDoc) *StatusDocResponse {
	return &StatusDocResponse{StatusDoc: statusDoc}
}

// Render processes responses before marshalling.
func (s *StatusDocResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
