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

// LicenseChecker is the structure passed to every checker method
type LicenseChecker struct {
	license   *lic.License
	statusDoc *lic.StatusDoc
}

// ErrResponse is the expected structure fetched from the LCP Server in cas of an error.
type ErrResponse struct {
	Type   string `json:"type"` // url
	Title  string `json:"title"`
	Status int    `json:"status"` // http status code
}

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

	c := LicenseChecker{}
	c.license = new(lic.License)

	// parse json data -> license
	err = json.Unmarshal(bytes, c.license)
	if err != nil {
		log.Errorf("Failed to unmarshal the license: %v", err)
		return err
	}

	err = c.ShowLicenseInfo()
	if err != nil {
		log.Errorf("Fatal error showing license info: %v", err)
		return err
	}

	// check the license
	err = c.CheckLicense(passphrase)
	if err != nil {
		log.Errorf("Fatal error checking the license: %v", err)
		return err
	}

	// checking the status document requires level 2+
	if level <= 1 {
		return nil
	}

	log.Info("-- Check the status document --")

	// get the license status
	err = c.GetStatusDoc()
	if err != nil {
		log.Errorf("Fata error getting the status document: %v", err)
		return err
	}

	// check the status document
	err = c.CheckStatusDoc()
	if err != nil {
		log.Errorf("Fatal error checking the status document: %v", err)
		return err
	}

	// checking the fresh license requires level 3+
	if level <= 2 {
		return nil
	}
	log.Info("-- Check the fresh license --")

	// get the fresh license
	err = c.GetFreshLicense()
	// no fatal error
	if err != nil {
		// check the fresh license
		err = c.CheckLicense(passphrase)
		if err != nil {
			log.Errorf("Fatal error checking the fresh license: %v", err)
			return err
		}
	}

	// updating the license requires level 4+
	if level <= 3 {
		return nil
	}
	log.Info("-- Check license updates --")

	// check updates to the license
	err = c.UpdateLicense()
	if err != nil {
		log.Errorf("Fatal error updating the license: %v", err)
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

// GetStatusDoc gets a license status from the URL passed in parameter
func (c *LicenseChecker) GetStatusDoc() error {

	// get the url of the license status
	var sdHref string
	for _, l := range c.license.Links {
		if l.Rel == "status" {
			sdHref = l.Href
		}
	}
	if sdHref == "" {
		return errors.New("the license is missing a link to a status document: stop testing")
	}

	// fetch the license status document
	newStatusDoc := new(lic.StatusDoc)
	err := getJson(sdHref, newStatusDoc)
	if err != nil {
		return err
	}

	// update the current status doc
	c.statusDoc = newStatusDoc
	return nil
}

// Get a fresh license from the provider system
func (c *LicenseChecker) GetFreshLicense() error {

	// get the url of the license
	var lHref string
	for _, s := range c.statusDoc.Links {
		if s.Rel == "license" {
			lHref = s.Href
		}
	}
	if lHref == "" {
		return errors.New("the status document is missing a link to a fresh license")
	}

	// fetch the fresh license
	freshLicense := new(lic.License)
	err := getJson(lHref, freshLicense)
	if err != nil {
		return err
	}

	// update the current license
	c.license = freshLicense
	return nil
}

// CheckResource verifies that the target of a link can be accessed
func CheckResource(href string) error {
	var expectedDuration time.Duration = 800 * time.Millisecond

	start := time.Now()
	// check that the resource can be fetched
	client := http.Client{
		Timeout: 2 * time.Second,
	}
	_, err := client.Head(href)
	if err != nil {
		return err
	}

	elapsed := time.Since(start)

	if elapsed > expectedDuration {
		log.Warningf("Access to %s took %s, which is quite long", href, elapsed)
	}
	return err
}

// getJson initializes any json struct with data fetched via http
func getJson(url string, target interface{}) error {

	client := http.Client{
		Timeout: 2 * time.Second,
	}
	r, err := client.Get(url)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	if r.StatusCode != 200 {
		log.Errorf("Error %d while fetching %s", r.StatusCode, url)

		// map the response to a error structure
		errResponse := new(ErrResponse)
		err := json.NewDecoder(r.Body).Decode(errResponse)
		if err != nil {
			log.Error("Invalid structure of the error response")
		} else {
			log.Infof("Server message: %s", errResponse.Title)
		}
		return errors.New("failed to fetch the resource")
	}

	return json.NewDecoder(r.Body).Decode(target)
}
