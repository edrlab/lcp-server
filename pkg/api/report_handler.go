// Copyright 2026 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package api

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/go-chi/render"
	log "github.com/sirupsen/logrus"
)

// ReportGeneratedLicenses generates a CSV report of licenses for a specific month or date
func (a *APICtrl) ReportGeneratedLicenses(w http.ResponseWriter, r *http.Request) {
	log.Debug("Report Generated Licenses, monthly or daily")

	var licenses *[]stor.LicenseInfo
	var err error
	var period string

	// Check for month parameter
	if month := r.URL.Query().Get("month"); month != "" {
		if date := r.URL.Query().Get("date"); date != "" {
			render.Render(w, r, ErrInvalidRequest(errors.New("cannot specify both month and date parameters")))
			return
		}
		licenses, err = a.Store.License().FindByDate(month, stor.IncludePubInfo)
		period = month
	} else if date := r.URL.Query().Get("date"); date != "" {
		licenses, err = a.Store.License().FindByDate(date, stor.IncludePubInfo)
		period = date
	} else {
		render.Render(w, r, ErrInvalidRequest(errors.New("missing required parameter: either month (YYYY-MM) or date (YYYY-MM-DD)")))
		return
	}

	if err != nil {
		render.Render(w, r, ErrServer(err))
		return
	}

	// Set CSV headers
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"licenses-report-%s.csv\"", url.QueryEscape(period)))

	// Create CSV writer
	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	// Write CSV header
	header := []string{"CreatedAt", "PublicationAltID", "PublicationTitle", "UserID", "Status", "Start", "End", "MaxEnd", "DeviceCount"}
	if err := csvWriter.Write(header); err != nil {
		log.Errorf("Error writing CSV header: %v", err)
		render.Render(w, r, ErrServer(err))
		return
	}

	// Write license data
	for _, license := range *licenses {
		record := []string{
			formatTimePtr(&license.CreatedAt),
			license.Publication.AltID,
			license.Publication.Title,
			license.UserID,
			license.Status,
			formatTimePtr(license.Start),
			formatTimePtr(license.End),
			formatTimePtr(license.MaxEnd),
			fmt.Sprintf("%d", license.DeviceCount),
		}

		if err := csvWriter.Write(record); err != nil {
			log.Errorf("Error writing CSV record: %v", err)
			render.Render(w, r, ErrServer(err))
			return
		}
	}
}

// formatTimePtr formats a time pointer to ISO 8601 string, returns empty string if nil
func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02T15:04:05Z07:00")
}