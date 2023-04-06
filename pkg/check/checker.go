// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"embed"
	"errors"
	"fmt"
	"net/http"
	"time"

	"encoding/json"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
	jsonschema "github.com/xeipuuv/gojsonschema"
)

//go:embed data/license.schema.json data/status.schema.json data/link.schema.json
var jsfs embed.FS

// Checker verifies a license, its license status and the fresh license.
// Parameters:
// bytes is a set of bytes representing the license
// passphrase is the license passphrase, which may be empty
// level is a level of tests.
// Access to the status document requires level 2 or upper.
// Modifications of the license require level 3 or upper.
func Checker(bytes []byte, passphrase string, level uint) error {

	log.Info("-- Check the license --")

	// check the validity of the license using the json schema
	var err error
	err = validateLicense(bytes)
	if err != nil {
		log.Errorf("Failed to validate the license: %v", err)
		return err
	}

	// parse json data -> license
	license := new(lic.License)
	err = json.Unmarshal(bytes, license)
	if err != nil {
		log.Errorf("Failed to unmarshal the license: %v", err)
		return err
	}

	// check the license
	err = CheckLicense(license, passphrase)
	if err != nil {
		log.Errorf("Failed to check the license: %v", err)
		return err
	}

	// checking the status document requires level 2+
	if level <= 1 {
		return nil
	}

	log.Info("-- Check the status document --")

	// get the license status
	var statusDoc *lic.StatusDoc
	statusDoc, err = getStatusDoc(license)
	if err != nil {
		log.Errorf("Failed to get the status document: %v", err)
		return err
	}

	// check the status document
	err = CheckStatusDoc(statusDoc)
	if err != nil {
		log.Errorf("Failed to check the status document: %v", err)
		return err
	}

	// checking the fresh license requires level 3+
	if level <= 2 {
		return nil
	}
	log.Info("-- Check the fresh license --")

	// get the fresh license
	freshLicense, err := getFreshLicense(statusDoc)
	if err != nil {
		log.Errorf("Failed to get the fresh license: %v", err)
		return err
	}

	// check the fresh license
	err = CheckLicense(freshLicense, passphrase)
	if err != nil {
		log.Errorf("Failed to check the fresh license: %v", err)
		return err
	}

	// updating the license requires level 4+
	if level <= 3 {
		return nil
	}
	log.Info("-- Update the license --")

	// check updates to the license
	err = UpdateLicense(freshLicense, statusDoc)
	if err != nil {
		log.Errorf("Failed to update the license: %v", err)
		return err
	}
	return nil
}

// Check the validity of the license using the JSON schema
func validateLicense(bytes []byte) error {

	licenseSchema, err := jsfs.ReadFile("data/license.schema.json")
	if err != nil {
		return err
	}
	linkSchema, err := jsfs.ReadFile("data/link.schema.json")
	if err != nil {
		return err
	}

	sl := jsonschema.NewSchemaLoader()
	linkLoader := jsonschema.NewStringLoader(string(linkSchema))
	err = sl.AddSchemas(linkLoader)
	if err != nil {
		return err
	}
	licenseLoader := jsonschema.NewStringLoader(string(licenseSchema))
	schema, err := sl.Compile(licenseLoader)
	if err != nil {
		return err
	}

	documentLoader := jsonschema.NewStringLoader(string(bytes[:]))

	result, err := schema.Validate(documentLoader)
	if err != nil {
		return err
	}

	if result.Valid() {
		log.Info("The license is valid vs the json schema")
	} else {
		log.Error("The license is invalid vs the json schema")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return errors.New("invalid license") // stop checking
	}
	return nil
}

// Get a license status from the URL passed in parameter
func getStatusDoc(license *lic.License) (*lic.StatusDoc, error) {

	// get the url of the license status
	var sdHref string
	for _, l := range license.Links {
		if l.Rel == "status" {
			sdHref = l.Href
		}
	}
	if sdHref == "" {
		return nil, errors.New("the license is missing a link to a status document: stop testing")
	}

	// fetch the license status document
	statusDoc := new(lic.StatusDoc)
	err := getJson(sdHref, statusDoc)
	if err != nil {
		return nil, err
	}
	return statusDoc, nil
}

// Get a fresh license from the provider system
func getFreshLicense(statusDoc *lic.StatusDoc) (*lic.License, error) {

	// get the url of the license
	var lHref string
	for _, s := range statusDoc.Links {
		if s.Rel == "license" {
			lHref = s.Href
		}
	}
	if lHref == "" {
		return nil, errors.New("the status document is missing a link to a fresh license: stop testing")
	}

	// fetch the fresh license
	license := new(lic.License)
	err := getJson(lHref, license)
	if err != nil {
		return nil, err
	}
	return license, nil
}

func getJson(url string, target interface{}) error {

	client := http.Client{
		Timeout: 2 * time.Second,
	}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	return json.NewDecoder(r.Body).Decode(target)
}
