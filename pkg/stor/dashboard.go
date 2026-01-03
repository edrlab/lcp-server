// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"time"
)

// DashboardData data model
type PublicationType struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type LicenseStatus struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

type ChartDataPoint struct {
	Month    string `json:"month"`
	Licenses int    `json:"licenses"`
}

type DashboardData struct {
	TotalPublications       int               `json:"totalPublications"`
	TotalUsers              int               `json:"totalUsers"`
	TotalLicenses           int               `json:"totalLicenses"`
	LicensesLast12Months    int               `json:"licensesLast12Months"`
	LicensesLastMonth       int               `json:"licensesLastMonth"`
	LicensesLastWeek        int               `json:"licensesLastWeek"`
	LicensesLastDay         int               `json:"licensesLastDay"`
	OldestLicenseDate       string            `json:"oldestLicenseDate"`
	LatestLicenseDate       string            `json:"latestLicenseDate"`
	OversharedLicensesCount int               `json:"oversharedLicensesCount"`
	PublicationTypes        []PublicationType `json:"publicationTypes"`
	LicenseStatuses         []LicenseStatus   `json:"licenseStatuses"`
	ChartData               []ChartDataPoint  `json:"chartData"`
}

// GetDashboard provides a summary of key metrics and statistics about the system.
func (s dashboardStore) GetDashboard(excessiveSharingThreshold int, limitToLast12Months bool) (*DashboardData, error) {
	var data DashboardData

	// Temporary variables for counts (GORM uses int64)
	var totalPublications, totalLicenses, totalUsers int64

	// Count total publications
	if err := s.db.Model(&Publication{}).Count(&totalPublications).Error; err != nil {
		return nil, err
	}
	data.TotalPublications = int(totalPublications)

	// Count total licenses
	if err := s.db.Model(&LicenseInfo{}).Count(&totalLicenses).Error; err != nil {
		return nil, err
	}
	data.TotalLicenses = int(totalLicenses)

	// Count unique users
	if err := s.db.Model(&LicenseInfo{}).Distinct("user_id").Count(&totalUsers).Error; err != nil {
		return nil, err
	}
	data.TotalUsers = int(totalUsers)

	// Dates for period calculations
	now := time.Now()
	last12Months := now.AddDate(-1, 0, 0)
	lastMonth := now.AddDate(0, -1, 0)
	lastWeek := now.AddDate(0, 0, -7)
	lastDay := now.AddDate(0, 0, -1)

	// Temporary variables for period counts
	var licensesLast12Months, licensesLastMonth, licensesLastWeek, licensesLastDay int64

	// Count licenses from the last 12 months
	if err := s.db.Model(&LicenseInfo{}).Where("created_at >= ?", last12Months).Count(&licensesLast12Months).Error; err != nil {
		return nil, err
	}
	data.LicensesLast12Months = int(licensesLast12Months)

	// Count licenses from the last month
	if err := s.db.Model(&LicenseInfo{}).Where("created_at >= ?", lastMonth).Count(&licensesLastMonth).Error; err != nil {
		return nil, err
	}
	data.LicensesLastMonth = int(licensesLastMonth)

	// Count licenses from the last week
	if err := s.db.Model(&LicenseInfo{}).Where("created_at >= ?", lastWeek).Count(&licensesLastWeek).Error; err != nil {
		return nil, err
	}
	data.LicensesLastWeek = int(licensesLastWeek)

	// Count licenses from the last day
	if err := s.db.Model(&LicenseInfo{}).Where("created_at >= ?", lastDay).Count(&licensesLastDay).Error; err != nil {
		return nil, err
	}
	data.LicensesLastDay = int(licensesLastDay)

	// Date of the oldest license
	var oldestLicense LicenseInfo
	if err := s.db.Model(&LicenseInfo{}).Order("created_at ASC").First(&oldestLicense).Error; err == nil {
		data.OldestLicenseDate = oldestLicense.CreatedAt.Format("2006-01-02")
	}

	// Date of the most recent license
	var latestLicense LicenseInfo
	if err := s.db.Model(&LicenseInfo{}).Order("created_at DESC").First(&latestLicense).Error; err == nil {
		data.LatestLicenseDate = latestLicense.CreatedAt.Format("2006-01-02")
	}

	// Count licenses with excessive sharing using configurable threshold
	var oversharedLicensesCount int64
	query := s.db.Model(&LicenseInfo{}).Where("device_count > ?", excessiveSharingThreshold)

	// Optionally limit to last 12 months
	if limitToLast12Months {
		query = query.Where("created_at >= ?", last12Months)
	}

	if err := query.Count(&oversharedLicensesCount).Error; err != nil {
		return nil, err
	}
	data.OversharedLicensesCount = int(oversharedLicensesCount)

	// Get publication types
	var pubTypes []struct {
		ContentType string
		Count       int64
	}
	if err := s.db.Model(&Publication{}).Select("content_type, count(*) as count").Group("content_type").Scan(&pubTypes).Error; err != nil {
		return nil, err
	}

	data.PublicationTypes = make([]PublicationType, len(pubTypes))
	for i, pt := range pubTypes {
		data.PublicationTypes[i] = PublicationType{
			Name:  mapContentTypeToDisplayName(pt.ContentType),
			Count: int(pt.Count),
		}
	}

	// Get license statuses
	var licenseStatuses []struct {
		Status string
		Count  int64
	}
	if err := s.db.Model(&LicenseInfo{}).Select("status, count(*) as count").Group("status").Scan(&licenseStatuses).Error; err != nil {
		return nil, err
	}

	data.LicenseStatuses = make([]LicenseStatus, len(licenseStatuses))
	for i, ls := range licenseStatuses {
		data.LicenseStatuses[i] = LicenseStatus{
			Name:  ls.Status,
			Count: int(ls.Count),
		}
	}

	// Chart data - licenses created per month for the last 12 months
	// Use a simpler approach that works across all database dialects
	// Get all licenses from the last 12 months and process them in Go
	var licenses []LicenseInfo
	if err := s.db.Model(&LicenseInfo{}).
		Select("created_at").
		Where("created_at >= ?", last12Months).
		Find(&licenses).Error; err != nil {
		return nil, err
	}

	// Process data in Go to create monthly chart data
	monthCounts := make(map[string]int)
	for _, license := range licenses {
		monthKey := license.CreatedAt.Format("2006-01")
		monthCounts[monthKey]++
	}

	// Convert to chart data format
	for monthKey, count := range monthCounts {
		if t, err := time.Parse("2006-01", monthKey); err == nil {
			data.ChartData = append(data.ChartData, ChartDataPoint{
				Month:    t.Format("Jan"),
				Licenses: count,
			})
		}
	}

	return &data, nil
}

// mapContentTypeToDisplayName converts MIME content types to human-readable display names
func mapContentTypeToDisplayName(contentType string) string {
	switch contentType {
	case "application/epub+zip":
		return "EPUB"
	case "application/pdf+lcp":
		return "PDF"
	case "application/audiobook+lcp":
		return "Audiobook"
	case "application/divina+lcp":
		return "Comics"
	default:
		return contentType // Return original if no mapping found
	}
}

type OversharedLicenseData struct {
	ID      string `json:"id"`
	PublicationID string `json:"publicationId"`
	AltID         string `json:"altId"`
	Title   string `json:"title"`
	UserID        string `json:"userId"`
	UserEmail     string `json:"userEmail"`
	Type    string `json:"type"`
	Status  string `json:"status"`
	Devices int    `json:"devices"`
}

// GetOversharedLicenses provides a list of licenses that have been shared across multiple devices.
func (s dashboardStore) GetOversharedLicenses(excessiveSharingThreshold int, limitToLast12Months bool) ([]OversharedLicenseData, error) {
	var licenses []OversharedLicenseData

	// Build the query with joins to get publication information
	query := s.db.Table("license_infos").
		Select(`
			license_infos.uuid as id,
			license_infos.publication_id as publication_id,
			publications.alt_id as alt_id,
			publications.title as title,
			license_infos.user_id as user_id,
			license_infos.user_email as user_email,
			CASE WHEN license_infos.end IS NULL THEN 'loan' ELSE 'buy' END as type,
			license_infos.status as status,
			license_infos.device_count as devices
		`).
		Joins("JOIN publications ON license_infos.publication_id = publications.uuid").
		Where("license_infos.device_count > ?", excessiveSharingThreshold)

	// Optionally limit to licenses created in the last 12 months
	if limitToLast12Months {
		last12Months := time.Now().AddDate(-1, 0, 0)
		query = query.Where("license_infos.created_at >= ?", last12Months)
	}

	// Execute the query
	if err := query.Scan(&licenses).Error; err != nil {
		return nil, err
	}

	return licenses, nil
}
