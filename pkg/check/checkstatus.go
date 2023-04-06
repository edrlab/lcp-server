// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"regexp"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
	jsonschema "github.com/xeipuuv/gojsonschema"
)

// Check the license status document
func CheckStatusDoc(statusDoc *lic.StatusDoc) error {

	// check that the status doc is valid vs the json schema
	err := validateStatusDoc(statusDoc)
	if err != nil {
		return err
	}

	// display the status of the license and the associated message
	log.Info("The status of the license is ", statusDoc.Status)
	if statusDoc.Message != "" {
		log.Info("Message: ", statusDoc.Message)
	}

	// check the link to the fresh license
	err = checkLicenseLink(statusDoc)
	if err != nil {
		return err
	}

	// check actionable links (register, renew, return)
	err = checkActionableLinks(statusDoc)
	if err != nil {
		return err
	}

	// display the max end date of the license
	if statusDoc.PotentialRights != nil && statusDoc.PotentialRights.End != nil {
		renewExt := *statusDoc.PotentialRights.End
		log.Infof("Potential renew extension: %s", renewExt.String())
	}

	// give info about events present in the status document
	dict := make(map[string]int)
	for _, ev := range statusDoc.Events {
		dict[ev.Type] = dict[ev.Type] + 1
	}
	log.Infof("%d events: %d register, %d renew, %d return", len(statusDoc.Events), dict["register"], dict["renew"], dict["return"])
	return nil
}

// Check the validity of the status doc using the JSON schema
func validateStatusDoc(statusDoc *lic.StatusDoc) error {

	// convert the status doc to a string
	bytes, err := json.Marshal(statusDoc)
	if err != nil {
		return err
	}

	// load the embedded schema
	statusDocSchema, err := jsfs.ReadFile("data/status.schema.json")
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
	statusDocLoader := jsonschema.NewStringLoader(string(statusDocSchema))
	schema, err := sl.Compile(statusDocLoader)
	if err != nil {
		return err
	}

	//docStr := string(bytes)
	//fmt.Println(docStr)

	documentLoader := jsonschema.NewStringLoader(string(bytes))

	// validate the status doc
	// TODO: it appears that the uri-template format used in links is not properly validated
	// using the current json schema package. We had to modify the link model in the schema
	// to get status documents validated. This json schema package is not maintained anymore,
	// threrefore we'll have to find a solution and propose a PR
	// to a maintained fork (https://github.com/gojsonschema/gojsonschema)
	result, err := schema.Validate(documentLoader)
	if err != nil {
		return err
	}

	if result.Valid() {
		log.Info("The status doc is valid vs the json schema")
	} else {
		log.Error("The status doc is invalid vs the json schema")
		for _, desc := range result.Errors() {
			fmt.Printf("- %s\n", desc)
		}
		return errors.New("invalid status doc") // stop checking
	}
	return nil
}

// Verifies the link to the fresh license
func checkLicenseLink(statusDoc *lic.StatusDoc) error {

	var licType, licHref string
	for _, s := range statusDoc.Links {
		if s.Rel == "license" {
			licType = s.Type
			licHref = s.Href
		}
	}
	if licHref == "" {
		log.Error("A status document must link to a fresh license")
	}
	if licType != "application/vnd.readium.lcp.license.v1.0+json" {
		log.Errorf("The mime type of the fresh license (%s) is invalid", licType)
	}

	// check that the fresh license can be fetched
	err := CheckResource(licHref)
	if err != nil {
		log.Errorf("The fresh license at %s is unreachable", licHref)
	}
	return nil
}

// Verifies actionable links
func checkActionableLinks(statusDoc *lic.StatusDoc) error {

	// compile the regexp for better perf
	regexpId, err := regexp.Compile(`\{\?.*id.*\}`)
	if err != nil {
		return err
	}
	regexpName, err := regexp.Compile(`\{\?.*name.*\}`)
	if err != nil {
		return err
	}

	hasRegister := false
	for _, s := range statusDoc.Links {

		switch s.Rel {
		case "license":
			return nil
		case "register":
			hasRegister = true
		case "renew":
			// a renew link may point at an html page
			if s.Type == "text/html" {
				log.Info("The renew link is referencing an html page")
				err := CheckResource(s.Href)
				if err != nil {
					log.Errorf("The renew page at %s is unreachable", s.Href)
				}
				return nil
			}
		case "return":
		default:
			log.Warningf("Unknown link type %s", s.Rel)
			return nil
		}
		// a register link is highly recommended in our implementation
		if !hasRegister {
			log.Warningf("A status document should have a register link")
		}
		// check the url
		_, err := url.Parse(s.Href)
		if err != nil {
			log.Errorf("The %s link must be expressed as a url", s.Rel)
		}
		// check that the link is a uri template
		if !s.Templated {
			log.Errorf("A %s link must be templated", s.Rel)
		}
		// check the presence of the id and name params in the uri template
		match := regexpId.Match([]byte(s.Href))
		if err != nil {
			return err
		}
		if !match {
			log.Errorf("Parameters id is missing in the register uri template")
		}
		match = regexpName.Match([]byte(s.Href))
		if err != nil {
			return err
		}
		if !match {
			log.Errorf("Parameters name is missing in the %s uri template", s.Rel)
		}
		// check the mime type of the link
		if s.Type != "application/vnd.readium.license.status.v1.0+json" {
			log.Errorf("The mime type of the %s link (%s) is invalid", s.Rel, s.Type)
		}
	}

	return nil
}
