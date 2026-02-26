// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"errors"
	"time"

	"github.com/go-playground/validator/v10"
	"gorm.io/gorm"
)

// LicenseInfo data model
// Note: the date of issue of the license is handled by the gorm model.
// but the license is not logically updated when a device registers,
// therefore we keep the Updated property, which must be maintained manually.
type LicenseInfo struct {
	gorm.Model
	CreatedAt     time.Time   `gorm:"index"`             // index on created_at, useful for dashboard queries
	Updated       *time.Time  `json:"updated,omitempty"` // see comment above
	UUID          string      `json:"uuid" validate:"required,uuid" gorm:"type:varchar(100);uniqueIndex"`
	Provider      string      `json:"provider" validate:"required,url" gorm:"type:varchar(255)"`
	UserID        string      `json:"user_id,omitempty" validate:"required" gorm:"type:varchar(100);index"`
	Start         *time.Time  `json:"start,omitempty"`
	End           *time.Time  `json:"end,omitempty"`
	MaxEnd        *time.Time  `json:"max_end,omitempty"`
	Copy          int32       `json:"copy,omitempty"`
	Print         int32       `json:"print,omitempty"`
	Status        string      `json:"status" validate:"oneof=ready active expired cancelled revoked" gorm:"type:varchar(100);index"`
	StatusUpdated *time.Time  `json:"status_updated,omitempty"`
	DeviceCount   int         `json:"device_count" gorm:"index"`
	PublicationID string      `json:"publication_id" validate:"required,uuid"  gorm:"type:varchar(100);index"` // implicit foreign key to the related publication
	Publication   Publication `gorm:"references:UUID" validate:"-"`                                            // the license belongs to the publication
}

// Validate checks required fields and values
func (l *LicenseInfo) Validate() error {

	validate := validator.New()
	return validate.Struct(l)
}

func (s licenseStore) ListAll() (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	// security: limited to 1000 results
	return &licenses, s.db.Limit(1000).Order("id DESC").Find(&licenses).Error
}

func (s licenseStore) List(pageNum, pageSize int) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	// pageNum starts at 1
	// result sorted to assure the same order for each request
	return &licenses, s.db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Order("id DESC").Find(&licenses).Error
}

func (s licenseStore) FindByUser(userID string, pubinfo bool) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	query := s.db.Limit(1000).Where("user_id= ?", userID).Order("license_infos.id DESC")
	
	if pubinfo {
		// Join with the publication table to get publication title
		query = query.Joins("Publication")
	}
	
	return &licenses, query.Find(&licenses).Error
}

func (s licenseStore) FindByPublication(publicationID string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("publication_id= ?", publicationID).Order("id DESC").Find(&licenses).Error
}

func (s licenseStore) FindByStatus(status string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("status= ?", status).Order("id DESC").Find(&licenses).Error
}

func (s licenseStore) FindByDeviceCount(min int, max int) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("device_count >= ? AND device_count <= ?", min, max).Order("id DESC").Find(&licenses).Error
}

// FindByDate finds licenses by creation date or month
// dateStr can be:
// - a specific date: "2024-02-15" (YYYY-MM-DD)
// - a month: "2024-02" (YYYY-MM)
func (s licenseStore) FindByDate(dateStr string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	
	// Check if it's a month (format: YYYY-MM) or a specific date (format: YYYY-MM-DD)
	if len(dateStr) == 7 { // Month format: YYYY-MM
		// Parse the month
		startOfMonth, err := time.Parse("2006-01", dateStr)
		if err != nil {
			return &licenses, err
		}
		// Get the start of next month
		startOfNextMonth := startOfMonth.AddDate(0, 1, 0)
		
		return &licenses, s.db.Limit(1000).
			Where("created_at >= ? AND created_at < ?", startOfMonth, startOfNextMonth).
			Order("id DESC").Find(&licenses).Error
	} else if len(dateStr) == 10 { // Date format: YYYY-MM-DD
		// Parse the specific date
		specificDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			return &licenses, err
		}
		// Get the start of next day
		startOfNextDay := specificDate.AddDate(0, 0, 1)
		
		return &licenses, s.db.Limit(1000).
			Where("created_at >= ? AND created_at < ?", specificDate, startOfNextDay).
			Order("id DESC").Find(&licenses).Error
	} else {
		return &licenses, errors.New("invalid date format: use YYYY-MM for month or YYYY-MM-DD for specific date")
	}
}

func (s licenseStore) Count() (int64, error) {
	var count int64
	return count, s.db.Model(LicenseInfo{}).Count(&count).Error
}

func (s licenseStore) Get(uuid string) (*LicenseInfo, error) {
	var license LicenseInfo
	return &license, s.db.Where("uuid = ?", uuid).First(&license).Error
}

func (s licenseStore) Create(newLicense *LicenseInfo) error {
	return s.db.Create(newLicense).Error
}

func (s licenseStore) Update(changedLicense *LicenseInfo) error {
	return s.db.Omit("Publication").Save(changedLicense).Error
}

func (s licenseStore) Delete(deletedLicense *LicenseInfo) error {
	return s.db.Delete(deletedLicense).Error
}
