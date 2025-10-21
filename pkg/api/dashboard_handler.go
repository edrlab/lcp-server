// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"net/http"

	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
)

// GetDashboardData provides a summary of key metrics and statistics about the system.
func (a *APICtrl) GetDashboardData(w http.ResponseWriter, r *http.Request) {

	var data *stor.DashboardData

	data, err := a.Store.Dashboard().GetDashboard(
		a.Config.Dashboard.ExcessiveSharingThreshold,
		a.Config.Dashboard.LimitToLast12Months,
	)
	if err != nil {
		log.Errorf("Get Dashboard Data: failed to get data: %v", err)
		render.Render(w, r, ErrServer(err))
		return
	}

	if err := render.Render(w, r, NewDashboardResponse(data)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// GetOversharedLicenses provides a list of licenses that have been shared across multiple devices.
func (a *APICtrl) GetOversharedLicenses(w http.ResponseWriter, r *http.Request) {

	var data []stor.OversharedLicenseData

	data, err := a.Store.Dashboard().GetOversharedLicenses(
		a.Config.Dashboard.ExcessiveSharingThreshold,
		a.Config.Dashboard.LimitToLast12Months,
	)
	if err != nil {
		log.Errorf("Get Overshared Licenses: failed to get data: %v", err)
		render.Render(w, r, ErrServer(err))
		return
	}

	if err := render.Render(w, r, NewOversharedLicensesResponse(data)); err != nil {
		render.Render(w, r, ErrRender(err))
	}
}

// --
// Request and Response payloads for the REST api.
// --

// DashboardResponse is the response payload for the dashboard.
type DashboardResponse struct {
	*stor.DashboardData
}

// NewDashboardResponse creates a rendered dashboard
func NewDashboardResponse(dashboard *stor.DashboardData) *DashboardResponse {
	return &DashboardResponse{DashboardData: dashboard}
}

// Render processes responses before marshalling.
func (s *DashboardResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

// OversharedLicensesResponse is the response payload for the overshared licenses.
type OversharedLicensesResponse []stor.OversharedLicenseData

// NewOversharedLicensesResponse creates a rendered overshared licenses response.
func NewOversharedLicensesResponse(licenses []stor.OversharedLicenseData) OversharedLicensesResponse {
	return OversharedLicensesResponse(licenses)
}

// Render processes responses before marshalling.
func (s OversharedLicensesResponse) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}
