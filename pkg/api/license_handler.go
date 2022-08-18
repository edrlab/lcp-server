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
	"time"

	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

// ListLicenses lists all licenses present in the database.
func (h *HandlerCtx) ListLicenses(w http.ResponseWriter, r *http.Request) {
	licenses, err := h.St.License().ListAll()
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseListResponse(licenses)); err != nil {
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
		licenses, err = h.St.License().FindByUser(userID)
		// by publication
	} else if pubID := r.URL.Query().Get("pub"); pubID != "" {
		licenses, err = h.St.License().FindByPublication(pubID)
		// by status
	} else if status := r.URL.Query().Get("status"); status != "" {
		licenses, err = h.St.License().FindByStatus(status)
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
		licenses, err = h.St.License().FindByDeviceCount(min, max)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseListResponse(licenses)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// CreateLicense adds a new License to the database.
func (h *HandlerCtx) CreateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	license := data.LicenseInfo

	// db create
	err := h.St.License().Create(license)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	if err := render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GetLicense returns a specific license
func (h *HandlerCtx) GetLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = h.St.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err := render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// UpdateLicense updates an existing License in the database.
func (h *HandlerCtx) UpdateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	license := data.LicenseInfo

	var currentLic *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		currentLic, err = h.St.License().Get(licenseID)
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
	err = h.St.License().Update(license)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	if err := render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// DeleteLicense removes an existing License from the database.
func (h *HandlerCtx) DeleteLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = h.St.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// db delete
	err = h.St.License().Delete(license)
	if err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// returning the deleted license to the caller allows for displaying useful info
	if err := render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// --
// Request and Response payloads for the REST api.
// --

// LicenseRequest is the request payload for licenses.
type LicenseRequest struct {
	*stor.LicenseInfo
}

// LicenseResponse is the response payload for licenses.
type LicenseResponse struct {
	*stor.LicenseInfo

	CreatedAt time.Time `json:"issued"` // overrride the property name
}

// NewLicenseListResponse creates a rendered list of licenses
func NewLicenseListResponse(licenses *[]stor.LicenseInfo) []render.Renderer {
	list := []render.Renderer{}
	for i := 0; i < len(*licenses); i++ {
		list = append(list, NewLicenseResponse(&(*licenses)[i]))
	}
	return list
}

// NewLicenseResponse creates a rendered license
func NewLicenseResponse(license *stor.LicenseInfo) *LicenseResponse {
	return &LicenseResponse{LicenseInfo: license}
}

// Bind post-processes requests after unmarshalling.
func (l *LicenseRequest) Bind(r *http.Request) error {
	if l.LicenseInfo == nil {
		return errors.New("missing required License payload")
	}
	// check required fields
	if l.UUID == "" {
		return errors.New("missing required UUID")
	}
	return nil
}

// Render processes responses before marshalling.
func (l *LicenseResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
