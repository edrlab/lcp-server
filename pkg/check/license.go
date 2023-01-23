// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"time"
)

// ====
// Definition of an LCP License
// ====

type License struct {
	Provider   string     `json:"provider"`
	UUID       string     `json:"id"`
	Issued     time.Time  `json:"issued"`
	Updated    *time.Time `json:"updated,omitempty"`
	Encryption Encryption `json:"encryption"`
	Links      *[]Link    `json:"links,omitempty"`
	User       UserInfo   `json:"user"`
	Rights     UserRights `json:"rights"`
	Signature  Signature  `json:"signature"`
}

type Encryption struct {
	Profile    string     `json:"profile,omitempty"`
	ContentKey ContentKey `json:"content_key,omitempty"`
	UserKey    UserKey    `json:"user_key"`
}

type ContentKey struct {
	Algorithm string `json:"algorithm,omitempty"`
	Value     []byte `json:"encrypted_value,omitempty"`
}

type UserKey struct {
	Algorithm string `json:"algorithm,omitempty"`
	TextHint  string `json:"text_hint,omitempty"`
	Keycheck  []byte `json:"key_check,omitempty"`
}

type UserInfo struct {
	ID        string   `json:"id"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

type Link struct {
	Rel       string `json:"rel"`
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Templated bool   `json:"templated,omitempty"`
	Size      int64  `json:"length,omitempty"`
	Checksum  string `json:"hash,omitempty"`
}

type UserRights struct {
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
	Print *int32     `json:"print,omitempty"`
	Copy  *int32     `json:"copy,omitempty"`
}

// ====
// Definition of an LCP License Status
// ====

type LicenseStatus struct {
	ID                int              `json:"-"`
	LicenseRef        string           `json:"id"`
	Status            string           `json:"status"`
	Updated           *Updated         `json:"updated,omitempty"`
	Message           string           `json:"message"`
	Links             []Link           `json:"links,omitempty"`
	DeviceCount       *int             `json:"device_count,omitempty"`
	PotentialRights   *PotentialRights `json:"potential_rights,omitempty"`
	Events            []Event          `json:"events,omitempty"`
	CurrentEndLicense *time.Time       `json:"-"`
}

type Updated struct {
	License *time.Time `json:"license,omitempty"`
	Status  *time.Time `json:"status,omitempty"`
}

type PotentialRights struct {
	End *time.Time `json:"end,omitempty"`
}

type Event struct {
	Type       string    `json:"type"`
	DeviceName string    `json:"name"`
	DeviceId   string    `json:"id"`
	Timestamp  time.Time `json:"timestamp"`
}

type Signature struct {
	Certificate []byte `json:"certificate"`
	Value       []byte `json:"value"`
	Algorithm   string `json:"algorithm"`
}
