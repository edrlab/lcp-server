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
	log "github.com/sirupsen/logrus"
)

// ListLicenses lists licenses present in the database.
func (a *APICtrl) ListLicenses(w http.ResponseWriter, r *http.Request) {
	log.Debug("List Licenses")

	page := r.Context().Value(PageKey).(int)
	perPage := r.Context().Value(PerPageKey).(int)
	var licenses *[]stor.LicenseInfo
	var err error

	if page == 0 || perPage == 0 {
		licenses, err = a.Store.License().ListAll()
	} else {
		licenses, err = a.Store.License().List(page, perPage)
	}
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseInfoListResponse(licenses)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// SearchLicenses searches licenses corresponding to a specific criteria.
func (a *APICtrl) SearchLicenses(w http.ResponseWriter, r *http.Request) {
	var licenses *[]stor.LicenseInfo
	var err error

	// search by user
	if userID := r.URL.Query().Get("user"); userID != "" {
		licenses, err = a.Store.License().FindByUser(userID)
		// by publication
	} else if pubID := r.URL.Query().Get("pub"); pubID != "" {
		licenses, err = a.Store.License().FindByPublication(pubID)
		// by status
	} else if status := r.URL.Query().Get("status"); status != "" {
		licenses, err = a.Store.License().FindByStatus(status)
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
		licenses, err = a.Store.License().FindByDeviceCount(min, max)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}
	if err := render.RenderList(w, r, NewLicenseInfoListResponse(licenses)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// CreateLicense adds a new license to the database.
func (a *APICtrl) CreateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseInfoRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	license := data.LicenseInfo

	// Check the presence of a UUID
	if license.UUID == "" {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required license UUID")))
		return
	}

	// force the status to ready (the caller does not has to set it)
	license.Status = stor.STATUS_READY

	// set the max end date if there is an end date and the max end date is not set in the input.
	// the renew max date will be 0 if not set in the configuration
	if license.End != nil && license.MaxEnd == nil {
		maxEnd := license.End.AddDate(0, 0, a.Config.Status.RenewMaxDays)
		license.MaxEnd = &maxEnd
	}

	// db create
	err := a.Store.License().Create(license)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	render.Status(r, http.StatusCreated)
	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GetLicense returns a specific license
func (a *APICtrl) GetLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = a.Store.License().Get(licenseID)
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
func (a *APICtrl) UpdateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	data := &LicenseInfoRequest{}
	if err := render.Bind(r, data); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}
	licUpdates := data.LicenseInfo

	var license *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = a.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// set updated fields
	license.Provider = licUpdates.Provider
	license.UserID = licUpdates.UserID
	license.Start = licUpdates.Start
	license.End = licUpdates.End
	license.MaxEnd = licUpdates.MaxEnd
	license.Copy = licUpdates.Copy
	license.Print = licUpdates.Print
	license.Status = licUpdates.Status
	license.StatusUpdated = licUpdates.StatusUpdated
	license.DeviceCount = licUpdates.DeviceCount

	// db update
	err = a.Store.License().Update(license)
	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	if err := render.Render(w, r, NewLicenseInfoResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// DeleteLicense removes an existing license from the database.
func (a *APICtrl) DeleteLicense(w http.ResponseWriter, r *http.Request) {

	var license *stor.LicenseInfo
	var err error

	// get the existing license
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		license, err = a.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrNotFound)
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// db delete
	err = a.Store.License().Delete(license)
	if err != nil {
		render.Render(w, r, ErrServer(err))
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
}

// LicenseInfoResponse is the response payload for licenses.
type LicenseInfoResponse struct {
	*stor.LicenseInfo
	// do not serialize the following properties
	//ID omit `json:"ID,omitempty"`
	//CreatedAt   omit `json:"CreatedAt,omitempty"`
	//UpdatedAt   omit `json:"UpdatedAt,omitempty"`
	//DeletedAt   omit `json:"DeletedAt,omitempty"`
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
