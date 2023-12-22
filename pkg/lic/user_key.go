// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"encoding/hex"
	"errors"
)

var LCP_PRODUCTION_LIB = false

// GenerateUserKey function prepares the user key
func GenerateUserKey(profile, passhash string) ([]byte, error) {

	if profile != "http://readium.org/lcp/basic-profile" {
		return nil, errors.New("this version can only process LCP basic profile; failed to decode the user passphrase")
	}
	// compute a byte array from a string
	value, err := hex.DecodeString(passhash)
	if err != nil {
		return nil, errors.New("failed to decode the user passphrase")
	}
	return value, nil
}
