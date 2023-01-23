// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"errors"
	"net/http"

	"github.com/edrlab/lcp-server/pkg/lic"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

type DeviceInfo struct {
	ID   string
	Name string
}

// Status returns a status document for the input license.
func (h *APIHandler) StatusDoc(w http.ResponseWriter, r *http.Request) {

	license, err := h.getLicenseInfo(w, r)
	if err != nil {
		return
	}

	err = returnStatusDoc(w, r, license)
	if err != nil {
		return
	}
}

// Register records a new device using the license and returns a status document.
func (h *APIHandler) Register(w http.ResponseWriter, r *http.Request) {

	// check the presence of the required params
	var lid string
	if lid = licenseID(w, r); lid == "" {
		return
	}
	if deviceInfo := deviceInfo(w, r); deviceInfo == nil {
		return
	}

	lm := lic.NewLicenseHandler(h.Config, h.Store)

	// register
	license, err := lm.Register(lid)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
	}

	err = returnStatusDoc(w, r, license)
	if err != nil {
		return
	}
}

// Return forces the expiration of a license and returns a status document.
func (h *APIHandler) Return(w http.ResponseWriter, r *http.Request) {

	license, err := h.getLicenseInfo(w, r)
	if err != nil {
		return
	}

	// update the license and add an event

	err = returnStatusDoc(w, r, license)
	if err != nil {
		return
	}
}

// Renew extends the lifetime of a license and returns a status document.
func (h *APIHandler) Renew(w http.ResponseWriter, r *http.Request) {

	license, err := h.getLicenseInfo(w, r)
	if err != nil {
		return
	}

	// update the license and add an event

	// if the request end date is > than potential end, extend to potential end

	// if the end date is already the potential end + renew request, error message =
	// "It is not possible to extend the end date of the license after February 3, 2020"
	// !! support accept-language in messages

	// see Thorium
	// https://github.com/readium/readium-desktop/blob/aacfe6bbd33db5623a01cfb2939713b6015c8790/src/main/services/lcp.ts#L341-L350
	// https://github.com/readium/readium-desktop/blob/aacfe6bbd33db5623a01cfb2939713b6015c8790/src/main/services/lcp.ts#L943-L953

	err = returnStatusDoc(w, r, license)
	if err != nil {
		return
	}
}

// --
// local functions
// --

func licenseID(w http.ResponseWriter, r *http.Request) (licenseID string) {

	if licenseID = chi.URLParam(r, "licenseID"); licenseID == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required license identifier")))
	}
	return
}

func deviceInfo(w http.ResponseWriter, r *http.Request) *DeviceInfo {

	var device DeviceInfo

	device.ID = chi.URLParam(r, "id")
	device.Name = chi.URLParam(r, "name")

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

func (h *APIHandler) getLicenseInfo(w http.ResponseWriter, r *http.Request) (*stor.LicenseInfo, error) {

	var license *stor.LicenseInfo
	var err error

	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = h.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required license identifier")))
		return nil, err
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return nil, err
	}
	return license, nil
}

func returnStatusDoc(w http.ResponseWriter, r *http.Request, license *stor.LicenseInfo) error {

	statusDoc := lic.NewStatusDoc(license)
	if err := render.Render(w, r, NewStatusDocResponse(statusDoc)); err != nil {
		render.Render(w, r, ErrRender(err))
		return err
	}
	return nil
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
