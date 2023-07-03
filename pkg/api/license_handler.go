// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"errors"
	"net/http"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

// GenerateLicense creates a license in the db and returns a fresh license
func (h *APIHandler) GenerateLicense(w http.ResponseWriter, r *http.Request) {

	// get the payload
	licRequest := &LicenseRequest{}
	if err := render.Bind(r, licRequest); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// get the corresponding publication
	var pubInfo *stor.Publication
	var err error
	if licRequest.PublicationID != "" {
		pubInfo, err = h.Store.Publication().Get(licRequest.PublicationID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication identifier in payload")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// set license info
	licInfo := newLicenseInfo(h.Config.License.Provider, licRequest)

	// store license info
	err = h.Store.License().Create(licInfo)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	// get back license info to retrieve gorm data
	licInfo, err = h.Store.License().Get(licInfo.UUID)
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	userInfo := lic.UserInfo{
		ID:        licRequest.UserID,
		Name:      licRequest.UserName,
		Email:     licRequest.UserEmail,
		Encrypted: licRequest.UserEncrypted,
	}
	encryption := lic.Encryption{
		Profile: licRequest.Profile,
		UserKey: lic.UserKey{
			TextHint: licRequest.TextHint,
		},
	}

	// generate the license
	license, err := lic.NewLicense(h.Config, h.Cert, pubInfo, licInfo, &userInfo, &encryption, licRequest.PassHash)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	if err = render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// GetFreshLicense returns a fresh license
func (h *APIHandler) GetFreshLicense(w http.ResponseWriter, r *http.Request) {
	var err error

	// get the payload
	licRequest := &LicenseRequest{}
	if err = render.Bind(r, licRequest); err != nil {
		render.Render(w, r, ErrInvalidRequest(err))
		return
	}

	// get the license
	var licInfo *stor.LicenseInfo
	if licenseID := chi.URLParam(r, "licenseID"); licenseID != "" {
		licInfo, err = h.Store.License().Get(licenseID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing licenseID parameter")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	// get the corresponding publication
	var pubInfo *stor.Publication

	if licInfo.PublicationID != "" {
		pubInfo, err = h.Store.Publication().Get(licInfo.PublicationID)
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required publication identifier in payload")))
		return
	}
	if err != nil {
		render.Render(w, r, ErrNotFound)
		return
	}

	userInfo := lic.UserInfo{
		ID:        licRequest.UserID,
		Name:      licRequest.UserName,
		Email:     licRequest.UserEmail,
		Encrypted: licRequest.UserEncrypted,
	}

	encryption := lic.Encryption{
		Profile: licRequest.Profile,
		UserKey: lic.UserKey{
			TextHint: licRequest.TextHint,
		},
	}

	// generate the license
	license, err := lic.NewLicense(h.Config, h.Cert, pubInfo, licInfo, &userInfo, &encryption, licRequest.PassHash)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
	if err := render.Render(w, r, NewLicenseResponse(license)); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

// newLicenseInfo sets license info from request parameters
func newLicenseInfo(provider string, licRequest *LicenseRequest) *stor.LicenseInfo {

	noLimit := int32(-1) // -1 stored for no print/copy limits
	if licRequest.Copy == nil {
		licRequest.Copy = &noLimit
	}
	if licRequest.Print == nil {
		licRequest.Print = &noLimit
	}

	licInfo := stor.LicenseInfo{
		UUID:          uuid.New().String(), // generate a random UUID
		Provider:      provider,
		UserID:        licRequest.UserID,
		PublicationID: licRequest.PublicationID,
		Start:         licRequest.Start,
		End:           licRequest.End,
		Copy:          *licRequest.Copy,
		Print:         *licRequest.Print,
		Status:        stor.STATUS_READY,
	}
	return &licInfo
}

// --
// Request and Response payloads for the REST api.
// --

// LicenseRequest is the request payload for licenses.
type LicenseRequest struct {
	PublicationID string     `json:"publication_id" validate:"required,uuid"`
	UserID        string     `json:"user_id,omitempty" validate:"required"`
	UserName      string     `json:"user_name,omitempty"`
	UserEmail     string     `json:"user_email,omitempty"`
	UserEncrypted []string   `json:"user_encrypted,omitempty"`
	Start         *time.Time `json:"start,omitempty"`
	End           *time.Time `json:"end,omitempty"`
	Copy          *int32     `json:"copy,omitempty"`
	Print         *int32     `json:"print,omitempty"`
	Profile       string     `json:"profile" validate:"required"`
	TextHint      string     `json:"text_hint" validate:"required"`
	PassHash      string     `json:"pass_hash" validate:"required"`
}

// Bind post-processes requests after unmarshalling.
func (l *LicenseRequest) Bind(r *http.Request) error {
	validate := validator.New()
	return validate.Struct(l)
}

// LicenseResponse is the response payload for licenses.
type LicenseResponse struct {
	*lic.License
}

// NewLicenseResponse creates a rendered license
func NewLicenseResponse(license *lic.License) *LicenseResponse {
	//lr := LicenseResponse{License: license}
	//fmt.Print(lr)
	return &LicenseResponse{License: license}
}

// Render processes responses before marshalling.
func (l *LicenseResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
