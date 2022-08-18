// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

// LicenseInfo data model
// Note: the date of issue of the license is handled by the gorm model.
// but the license is not logically updated when the device count is updated,
// therefore we keep the Updated property.
type LicenseInfo struct {
	gorm.Model
	UUID          string          `json:"uuid" gorm:"uniqueIndex"`
	Updated       *time.Time      `json:"updated,omitempty"`
	UserID        string          `json:"user_id" gorm:"index"`
	Provider      string          `json:"provider"`
	Start         *time.Time      `json:"start,omitempty"`
	End           *time.Time      `json:"end,omitempty"`
	Copy          uint32          `json:"copy,omitempty"`
	Print         uint32          `json:"print,omitempty"`
	Status        string          `json:"status" gorm:"index"`
	StatusUpdated *time.Time      `json:"status_updated,omitempty"`
	DeviceCount   int             `json:"device_count"`
	PublicationID string          `json:"publication_id"`  //  foreign key to the attached publication
	Publication   PublicationInfo `gorm:"references:UUID"` // attached publication
}

// Validate checks required fields and values
func (l *LicenseInfo) Validate() error {

	if l.UUID == "" {
		return errors.New("required field missing: UUID")
	}
	if l.UserID == "" {
		return errors.New("required field missing: UserID")
	}
	if l.PublicationID == "" {
		return errors.New("required Publication missing")
	}
	return nil
}

func (s licenseStore) ListAll() (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	// security: limited to 1000 results
	return &licenses, s.db.Limit(1000).Order("id ASC").Find(&licenses).Error
}

func (s licenseStore) List(pageSize, pageNum int) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	// pageNum starts at 1
	// result sorted to assure the same order for each request
	return &licenses, s.db.Offset((pageNum - 1) * pageSize).Limit(pageSize).Order("id ASC").Find(&licenses).Error
}

func (s licenseStore) FindByUser(userID string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("user_id= ?", userID).Find(&licenses).Error
}

func (s licenseStore) FindByPublication(publicationID string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("publication_id= ?", publicationID).Find(&licenses).Error
}

func (s licenseStore) FindByStatus(status string) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("status= ?", status).Find(&licenses).Error
}

func (s licenseStore) FindByDeviceCount(min int, max int) (*[]LicenseInfo, error) {
	licenses := []LicenseInfo{}
	return &licenses, s.db.Limit(1000).Where("device_count >= ? AND device_count <= ?", min, max).Find(&licenses).Error
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
