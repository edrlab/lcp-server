// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package stor

import (
	"time"
)

// Event data model
type Event struct {
	ID         string    `json:"-"`
	LicenseID  string    `json:"-"`
	Timestamp  time.Time `json:"timestamp"`
	Type       string    `json:"type"`
	DeviceName string    `json:"name"`
	DeviceId   string    `json:"id"`
}
