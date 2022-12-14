// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListLicenses lists all licenses present in the database.
func (h *HandlerCtx) ListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := h.Store.License().ListAll()
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseInfoListResponse(licenses)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// SearchLicenses searches licenses corresponding to a specific criteria.
func (h *HandlerCtx) SearchLicenses(w http.ResponseWriter, r *http.Request) {
	var licenses *[]stor.LicenseInfo
	var err error

	// search by user
	if userID := r.URL.Query().Get("user"); userID != "" {
		licenses, err = h.Store.License().FindByUser(userID)
		// by publication
	} else if pubID := r.URL.Query().Get("pub"); pubID != "" {
		licenses, err = h.Store.License().FindByPublication(pubID)
		// by status
	} else if status := r.URL.Query().Get("status"); status != "" {
		licenses, err = h.Store.License().FindByStatus(status)
		// by count
	} else if count := r.URL.Query().Get("count"); count != "" {
		// count is a "min:max" tuple
		var min, max int
		parts := strings.Split(count, ":")
		if len(parts) != 2 {
			render.Render(w, r, ErrInvalidRequest(fmt.Errorf("invalid count parameter: %s", count)))
			return
		}
		if min, err = strconv.Atoi(parts[0]); err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
		}
		if max, err = strconv.Atoi(parts[1]); err != nil {
			render.Render(w, r, ErrInvalidRequest(err))
		}
		licenses, err = h.Store.License().FindByDeviceCount(min, max)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseInfoListResponse(licenses)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// CreateLicense adds a new license to the database.
func (h *HandlerCtx) CreateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseInfoRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	license := data.LicenseInfo
	if license.Status != "ready" {
		license.Status = "ready" // force the status
	}

	// db create
	err := h.Store.License().Create(license)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GetLicense returns a specific license
func (h *HandlerCtx) GetLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = h.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required license identifier")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// UpdateLicense updates an existing License in the database.
func (h *HandlerCtx) UpdateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseInfoRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	license := data.LicenseInfo

	var currentLic *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		currentLic, err = h.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// set the gorm fields
	license.ID = currentLic.ID
	license.CreatedAt = currentLic.CreatedAt
	license.UpdatedAt = currentLic.UpdatedAt
	license.DeletedAt = currentLic.DeletedAt

	// db update
	err = h.Store.License().Update(license)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// DeleteLicense removes an existing license from the database.
func (h *HandlerCtx) DeleteLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = h.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// db delete
	err = h.Store.License().Delete(license)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// returning the deleted license to the caller allows for displaying useful info
	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// --
// Request and Response payloads for the REST api.
// --

// LicenseInfoRequest is the request payload for licenses.
type LicenseInfoRequest struct {
	*stor.LicenseInfo
	ID        omit `json:"ID,omitempty"`
	CreatedAt omit `json:"CreatedAt,omitempty"`
	UpdatedAt omit `json:"UpdatedAt,omitempty"`
	DeletedAt omit `json:"DeletedAt,omitempty"`
}

// LicenseInfoResponse is the response payload for licenses.
type LicenseInfoResponse struct {
	*stor.LicenseInfo
	DeletedAt   omit `json:"DeletedAt,omitempty"`
	Publication omit `json:"Publication,omitempty"`
}

// NewLicenseInfoListResponse creates a rendered list of licenses
func NewLicenseInfoListResponse(licenses *[]stor.LicenseInfo) []render.Renderer {
	list := []render.Renderer{}
	for i := 0; i < len(*licenses); i++ {
		list = append(list, NewLicenseInfoResponse(&(*licenses)[i]))
	}
	return list
}

// NewLicenseInfoResponse creates a rendered license
func NewLicenseInfoResponse(license *stor.LicenseInfo) *LicenseInfoResponse {
	return &LicenseInfoResponse{LicenseInfo: license}
}

// Bind post-processes requests after unmarshalling.
func (l *LicenseInfoRequest) Bind(r *http.Request) error {
	return l.LicenseInfo.Validate()
}

// Render processes responses before marshalling.
func (l *LicenseInfoResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
