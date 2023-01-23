// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

// Update a license using register / renew / return features
func UpdateLicense(license License, licenseStatus LicenseStatus) error {

	// check that the license is not in a final state (expired / revoked)

	// check that the device cannot be registered with empty id & name params

	// check register
	var err error
	err = checkRegister(licenseStatus)
	if err != nil {
		return err
	}

	// check renew
	err = checkRenew(license, licenseStatus)
	if err != nil {
		return err
	}

	// check return
	err = checkReturn(license, licenseStatus)
	if err != nil {
		return err
	}
	return nil
}

// Check register features
func checkRegister(licenseStatus LicenseStatus) error {

	// request registering the device

	// check errors

	// check that the status document which is returned is valid vs the json schema

	// check that the timestamp of the status document has been updated

	// check the new status of the license
	// if the status was ready, it must now be active.

	// test if a register event has been added to the status document

	return nil
}

// Check renew features
func checkRenew(license License, licenseStatus LicenseStatus) error {

	// test if the license can be extended

	// check if the extension can be done via the API (http put)

	// request an extension of the license (before the max end date)

	// check errors

	// check that the status document which is returned is valid vs the json schema

	// display the new status of the license

	// check that the timestamp of the status document has been updated

	// test if a renew event has been added to the status document

	// fetch the fresh license and check that it has been correctly updated

	// request an extension with an incorrect timestamp
	// and check that the server responds with an error

	// request an extension of the license after the max end date
	// and check that the server responds with an error

	return nil
}

// Check return features
func checkReturn(license License, licenseStatus LicenseStatus) error {

	// test if the license can be returned

	// request the return of the license

	// check errors

	// check that the status document which is returned is valid vs the json schema

	// check the new status of the license

	// check that the timestamp of the status document has been updated

	// test if a return event has been added to the status document

	// fetch the fresh license and check that it has been correctly updated
	// the end date must now be before now

	return nil
}
