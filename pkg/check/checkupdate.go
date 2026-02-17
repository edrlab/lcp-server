// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
)

// This device ID is used for register, renew and return.
// It is generated each time the checker runs.
var testDeviceId string

// Update a license using register / renew / return features
func (c *LicenseChecker) UpdateLicense() error {

	// check that the license is not in a final state
	if c.statusDoc.Status == "expired" {
		log.Infof("It is not possible to update a license which has expired")
		return nil
	}
	if c.statusDoc.Status == "revoked" || c.statusDoc.Status == "returned" || c.statusDoc.Status == "cancelled" {
		log.Infof("It is not possible to update a license which has been %s", c.statusDoc.Status)
		return nil
	}

	// check register
	var err error
	err = c.CheckRegister()
	if err != nil {
		return err
	}

	// 1sc pause
	time.Sleep(time.Second)

	// check renew
	err = c.CheckRenew()
	if err != nil {
		return err
	}

	// 1sc pause
	time.Sleep(time.Second)

	// check return
	err = c.CheckReturn()
	if err != nil {
		return err
	}
	return nil
}

// CheckRegister verifies register features
func (c *LicenseChecker) CheckRegister() error {

	log.Info("Checking license registration ...")

	// select the register link
	registerLink := c.GetStatusLink("register")
	if registerLink == nil {
		return errors.New("missing register link")
	}

	// set the link with a random device id and name
	url, err := setLinkUrl(registerLink.Href)
	if err != nil {
		return err
	}

	// request registering the device
	r, err := http.Post(url, "", nil)
	if err != nil {
		return err
	}
	defer r.Body.Close()

	// process the response
	err = c.ProcessResponse(r)
	if err != nil {
		return err
	}

	// check the new status of the license
	if c.statusDoc.Status != "active" {
		log.Errorf("The new status should be active, not %s", c.statusDoc.Status)
	}
	log.Info("License registration successful")
	return nil
}

// CheckRenew verifies renew features
func (c *LicenseChecker) CheckRenew() error {

	log.Info("Checking license extension (renewal) ...")

	// test if the license can be extended
	if c.license.Rights.End == nil {
		log.Infof("This license has no end date; won't check renew")
		return nil
	}

	// select the renew link
	renewLink := c.GetStatusLink("renew")
	if renewLink == nil {
		log.Infof("The status document has no renew link; won't check renew")
		return nil
	}

	// we cannot test a renew link based on a web page
	if renewLink.Type == "text/html" {
		log.Info("The renew link references a web page; won't check renew")
		return nil
	}

	// check the current status of the license
	if c.statusDoc.Status != "active" {
		log.Infof("The license is not in active state; won't try to extend it")
		return nil
	}

	// set the link with a test id, name (but no explicit end date)
	url, err := setLinkUrl(renewLink.Href)
	if err != nil {
		return nil
	}

	if c.statusDoc.PotentialRights == nil || c.statusDoc.PotentialRights.End == nil {
		log.Warning("The license has no max end date, unlimited renewal is allowed")
	} else {
		log.Info("The max end date for renewal is ", c.statusDoc.PotentialRights.End.Format(time.RFC822))
	}

	log.Info("Requesting extension with no end date ...")

	// request renewing the license
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return nil
	}
	r, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer r.Body.Close()

	// process the response
	err = c.ProcessResponse(r)
	// let's continue the tests if there was an error
	// especially if there was an attempt to renew with a date before the current end date
	if err == nil {
		// check the new status of the license
		if c.statusDoc.Status != "active" {
			log.Error("The new status should be active, not ", c.statusDoc.Status)
		} else {
			log.Info("License extension successful")
		}

		// 1/2sc pause
		time.Sleep(500 * time.Millisecond)

		// fetch the fresh license and check that it has been correctly updated
		err = c.GetFreshLicense()
		if err != nil {
			log.Error("Failed to get the fresh license: ", err)
			return nil
		} else {
			// the fresh license must have an update timestamp
			if c.license.Updated == nil {
				log.Error("The fresh license update timestamp is absent")
				return nil
			}
			// if the license time has been updated, it was necessarily during the last 2 seconds
			freshUpdate := time.Now().Add(-2 * time.Second)
			if c.license.Updated.Before(freshUpdate) {
				log.Error("The fresh license update timestamp was not properly updated")
			}
			log.Info("The license end timestamp is now ", c.license.Rights.End.Truncate(time.Second).Format(time.RFC822))
		}
	}

	// if there is no max end date, do not request additional extensions
	if c.statusDoc.PotentialRights == nil || c.statusDoc.PotentialRights.End == nil {
		return nil
	}

	// 1/2sc pause
	time.Sleep(500 * time.Millisecond)

	// request an extension of the license, again
	// this time with an explicit end date to the license, one day before the max end date
	end := c.statusDoc.PotentialRights.End.Add(-24 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	log.Info("Requesting extension with an explicit end date: ", end)

	url2 := url + "&end=" + end
	req, err = http.NewRequest("PUT", url2, nil)
	if err != nil {
		return nil
	}
	r, err = client.Do(req)
	if err != nil {
		return nil
	}
	err = c.ProcessResponse(r)
	if err == nil {
		log.Info("License extension successful")
	}

	// request an extension with an incorrect timestamp
	// and check that the server responds with an error
	log.Info("Requesting extension with an incorrect end date: 2000")

	url3 := url + "&end=2000"
	req, err = http.NewRequest("PUT", url3, nil)
	if err != nil {
		return nil
	}
	r, err = client.Do(req)
	if err != nil {
		return nil
	}
	err = c.ProcessResponse(r)
	if err == nil {
		log.Error("License extension with a wrong end date should fail")
	}

	// 1/2sc pause
	time.Sleep(500 * time.Millisecond)

	// request an extension of the license after the max end date
	// and check that the server responds with an error
	end = c.statusDoc.PotentialRights.End.Add(48 * time.Hour).Truncate(time.Second).Format("2006-01-02T15:04:05Z")
	log.Info("Requesting extension beyond the max datetime limit:", end)

	url4 := url + "&end=" + end
	req, err = http.NewRequest("PUT", url4, nil)
	if err != nil {
		return nil
	}
	r, err = client.Do(req)
	if err != nil {
		return nil
	}
	err = c.ProcessResponse(r)
	if err == nil {
		log.Info("License extension with an end date after the allowed max should fail")
	}

	return nil
}

// CheckReturn verifies return features
func (c *LicenseChecker) CheckReturn() error {

	log.Info("Checking license return ...")

	// test if the license can be returned
	if c.license.Rights.End == nil {
		log.Infof("This license has no end date; won't check return")
		return nil
	}

	// select the return link
	returnLink := c.GetStatusLink("return")
	if returnLink == nil {
		log.Infof("The status document has no return link; won't check return")
		return nil
	}

	// check the current status of the license
	if c.statusDoc.Status != "active" {
		log.Infof("The license is not in active state; won't try to return it")
		return nil
	}

	// set the link with a test id and name
	url, err := setLinkUrl(returnLink.Href)
	if err != nil {
		return err
	}

	// init a returned time to check that the license is correctly updated after the return
	returnTime := time.Now()

	// request the return of the license
	client := &http.Client{}
	req, err := http.NewRequest("PUT", url, nil)
	if err != nil {
		return nil
	}
	r, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer r.Body.Close()

	// process the response
	err = c.ProcessResponse(r)
	if err != nil {
		log.Info("Return didn't succeed: ", err)
		return nil
	}

	// check the new status of the license
	if c.statusDoc.Status != "returned" {
		log.Error("The new status should be returned, not ", c.statusDoc.Status)
	} else {
		log.Info("License return was successful")
	}

	// 1/2sc pause
	time.Sleep(500 * time.Millisecond)

	// fetch the fresh license and check that it has been correctly updated
	err = c.GetFreshLicense()
	if err != nil {
		log.Warning("Failed to get the fresh license after it was returned. This is not an error: ", err)
		return nil
	} else {
		if c.license.Updated == nil {
			log.Error("The fresh license update timestamp is absent")
			return nil
		}
		if c.license.Rights.End == nil {
			log.Error("The fresh license end timestamp is absent")
			return nil
		}
		log.Info("The license end timestamp is now ", c.license.Rights.End.Format(time.RFC822))
		// takes into account a possible small difference of time between the server and the checker
		if c.license.Updated.Before(returnTime.Add(-2 * time.Minute)) {
			log.Error("The fresh license update timestamp was not properly updated")
		}
		if c.license.Rights.End.After(time.Now().Add(2 * time.Minute)) {
			log.Error("The fresh license end timestamp was not properly updated")
		}
	}

	return nil
}

// get a link by its name
func (c *LicenseChecker) GetStatusLink(linkRel string) *lic.Link {

	// select the link
	var link lic.Link
	var found bool
	for _, link = range c.statusDoc.Links {
		if link.Rel == linkRel {
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	return &link
}

// ProcessResponse processes the response to a register, renew or return request
func (c *LicenseChecker) ProcessResponse(r *http.Response) error {

	if r.StatusCode != 200 {
		log.Warningf("The server returned an error %d", r.StatusCode)

		// map the response to an error structure
		errResponse := new(ErrResponse)
		err := json.NewDecoder(r.Body).Decode(errResponse)
		if err != nil {
			log.Error("Invalid structure of the error response")
		} else if errResponse.Detail != "" {
			log.Infof("Server message: %s", errResponse.Detail)
		} else {
			log.Infof("Server message, title: %s", errResponse.Title)
		}
		return errors.New("command failed")
	}

	// get the new status document
	newStatusDoc := new(lic.StatusDoc)
	err := json.NewDecoder(r.Body).Decode(newStatusDoc)
	if err != nil {
		log.Error("Invalid updated status doc structure")
		return err
	}

	// check that the timestamp of the status document has been updated
	if !newStatusDoc.Updated.Status.After(c.statusDoc.Updated.Status) {
		log.Errorf("The status doc timestamp has not been updated")
	}
	// test if an event has been added to the status document
	if len(newStatusDoc.Events) != len(c.statusDoc.Events)+1 {
		log.Errorf("A new event should have been created")
	} else {
		lastIndex := len(newStatusDoc.Events) - 1
		log.Infof("A new event was created, of type %s", newStatusDoc.Events[lastIndex].Type)
	}

	// update the current status doc
	c.statusDoc = newStatusDoc

	return nil
}

// setLinkUrl replaces the param of a templated URL by test params
func setLinkUrl(templatedUrl string) (url string, err error) {

	// the template may use the '?' or '&' form
	rSet := [2]string{`\{\?.*\}`, `\{\&.*\}`}
	// init the global test device id
	if testDeviceId == "" {
		testDeviceId = "lcp-checker-" + strconv.Itoa(rand.Intn(100))
		log.Infof("The test device ID is %s", testDeviceId)

	}
	param1 := fmt.Sprintf("?id=%s&name=%s", testDeviceId, testDeviceId)
	param2 := fmt.Sprintf("&id=%s&name=%s", testDeviceId, testDeviceId)
	params := [2]string{param1, param2}
	var rx *regexp.Regexp
	for idx, r := range rSet {
		rx, err = regexp.Compile(r)
		if err != nil {
			return
		}
		match := rx.Match([]byte(templatedUrl))
		if match {
			// set id and name params
			url = string(rx.ReplaceAll([]byte(templatedUrl), []byte(params[idx])))
			return
		}
	}
	return
}
