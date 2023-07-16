// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"time"
)

// Event data model
// we don't include the full gorm model here, has no update nor soft deletion occurs on events
type Event struct {
	ID         uint        `json:"-" gorm:"primaryKey"`
	Timestamp  time.Time   `json:"timestamp"`
	Type       string      `json:"type"`
	DeviceName string      `json:"name"`
	DeviceID   string      `json:"id" gorm:"index"`
	LicenseID  string      `json:"-"  gorm:"index"`          // implicit foreign key to the related license
	License    LicenseInfo `json:"-" gorm:"references:UUID"` // the event belongs to the license
}

func (s eventStore) List(licenseID string) (*[]Event, error) {
	events := []Event{}
	// security: limited to 500 results
	return &events, s.db.Limit(500).Where("license_id= ?", licenseID).Order("id ASC").Find(&events).Error
}

func (s eventStore) GetRegisterByDevice(licenseID string, deviceID string) (*Event, error) {
	var event Event
	return &event, s.db.Where("license_id= ? and type= 'register' and device_id= ?", licenseID, deviceID).First(&event).Error
}

func (s eventStore) Count(licenseID string) (int64, error) {
	var count int64
	return count, s.db.Model(Event{}).Count(&count).Error
}

func (s eventStore) Get(id uint) (*Event, error) {
	var event Event
	return &event, s.db.Where("id = ?", id).First(&event).Error
}

func (s eventStore) Create(newEvent *Event) error {
	return s.db.Create(newEvent).Error
}

func (s eventStore) Update(changedEvent *Event) error {
	return s.db.Omit("License").Save(changedEvent).Error
}

func (s eventStore) Delete(deletedEvent *Event) error {
	return s.db.Delete(deletedEvent).Error
}
