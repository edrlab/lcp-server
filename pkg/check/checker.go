// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"embed"
	"errors"
	"fmt"

	"encoding/json"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
	jsonschema "github.com/xeipuuv/gojsonschema"
)

// Checker verifies a license, its license status and the fresh license.
// Parameters:
// bytes is a set of bytes representing the license
// passphrase is the license passphrase, which may be empty
// level is a level of tests.
// Access to the status document requires level 2 or upper.
// Modifications of the license require level 3 or upper.
func Checker(bytes []byte, passphrase string, level uint) error {

	// check the validity of the license using the json schema
	err := validateLicense(bytes)
	if err != nil {
		return err
	}

	// parse json data -> license
	var license lic.License
	err = json.Unmarshal(bytes, &license)
	if err != nil {
		return err
	}

	// check the license
	err = CheckLicense(license, passphrase)
	if err != nil {
		return err
	}

	// no access to the the status document at level 0 or 1
	if level <= 1 {
		return nil
	}

	// get the license status
	var statusDoc lic.StatusDoc
	statusDoc, err = GetStatusDoc(license)
	if err != nil {
		return err
	}

	// check the status document
	err = CheckStatusDoc(statusDoc)
	if err != nil {
		return err
	}

	// get the fresh license
	var freshLicense lic.License
	freshLicense, err = GetFreshLicense(statusDoc)
	if err != nil {
		return err
	}

	// check the fresh license
	err = CheckLicense(freshLicense, passphrase)
	if err != nil {
		return err
	}

	// no modification of the license at level 2
	if level <= 2 {
		return nil
	}

	// check updates to the license
	err = UpdateLicense(freshLicense, statusDoc)
	if err != nil {
		return err
	}
	//

	return nil
}

//go:embed data/license.schema.json
var lsf embed.FS

// Check the validity of the license using the JSON schema
func validateLicense(bytes []byte) error {

	log.Info("Validating the license")

	schema, err := lsf.ReadFile("data/license.schema.json")
	if err != nil {
		return err
	}

	schemaLoader := jsonschema.NewStringLoader(string(schema))
	documentLoader := jsonschema.NewStringLoader(string(bytes[:]))

	result, err := jsonschema.Validate(schemaLoader, documentLoader)
	if err != nil {
		return err
	}

	if result.Valid() {
		log.Info("The license is valid")
	} else {
		log.Error("The license is invalid. see errors :")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return errors.New("invalid license (/ json schema)")
	}
	return nil
}

// Get a license status from the URL passed in parameter
func GetStatusDoc(license lic.License) (lic.StatusDoc, error) {
	var statusDoc lic.StatusDoc

	// get the url of the license status

	// fetch the license status document

	return statusDoc, nil
}

// Get a fresh license from the provider system
func GetFreshLicense(statusDoc lic.StatusDoc) (lic.License, error) {
	var license lic.License

	// get the url of the license

	// fetch the license document

	return license, nil
}
