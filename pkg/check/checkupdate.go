// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"encoding/json"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
)

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

	// tiny pause
	time.Sleep(time.Second)

	// check renew
	err = c.CheckRenew()
	if err != nil {
		return err
	}

	// tiny pause
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

	// check the current status of the license
	if c.statusDoc.Status != "ready" {
		log.Infof("The license is not is ready state; wont' try to register")
		return nil
	}

	// set the link with a test id and name
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

	log.Info("The max end date is ", c.statusDoc.PotentialRights.End.Format(time.RFC822))

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
	if err != nil {
		log.Info("Renew with an empty end date didn't succeed: ", err)
		return nil
	}

	// check the new status of the license
	if c.statusDoc.Status != "active" {
		log.Error("The new status should be active, not ", c.statusDoc.Status)
	}

	// fetch the fresh license and check that it has been correctly updated
	err = c.GetFreshLicense()
	if err != nil {
		log.Error("Failed to get the fresh license: ", err)
		return nil
	} else {
		freshUpdate := time.Now().Add(-1 * time.Minute)
		if c.license.Updated.Before(freshUpdate) {
			log.Error("The fresh license update timestamp was not properly updated")
		}
		log.Info("The license end timestamp is now ", c.license.Rights.End.Truncate(time.Second).Format(time.RFC822))
	}

	// request an extension of the license, again
	// this time with an explicit end date to the license, one day before the max end date
	end := c.statusDoc.PotentialRights.End.Add(-24 * time.Hour).Truncate(time.Second).Format(time.RFC3339)
	log.Info("Requesting extension with an explicit end date: ", c.statusDoc.PotentialRights.End.Add(-24*time.Hour).Truncate(time.Second).Format(time.RFC822))

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
	if err != nil {
		log.Info("Renew with an explicit end date didn't succeed: ", err)
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
	if err != nil {
		log.Info("Renew with a wrong end date didn't succeed: ", err)
	}

	// request an extension of the license after the max end date
	// and check that the server responds with an error
	end = c.statusDoc.PotentialRights.End.Add(48 * time.Hour).Truncate(time.Second).Format("2006-01-02T15:04:05")
	log.Info("Requesting extension with an overlong end date:")

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
	if err != nil {
		log.Info("Renew with an end date after the allowed max didn't succeed: ", err)
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
	}

	// fetch the fresh license and check that it has been correctly updated
	err = c.GetFreshLicense()
	if err != nil {
		log.Error("Failed to get the fresh license: ", err)
		return nil
	} else {
		log.Info("The license end timestamp is now ", c.license.Rights.End.Format(time.RFC822))
		freshUpdate := time.Now().Add(-1 * time.Minute)
		if c.license.Updated.Before(freshUpdate) {
			log.Error("The fresh license update timestamp was not properly updated")
		}
		if c.license.Rights.End.After(time.Now()) {
			log.Error("The fresh license end timestamp was not properly updated")
		}
	}

	return nil
}

// get a link by its name
func (c *LicenseChecker) GetStatusLink(linkRel string) *lic.Link {

	// select the register link
	var link lic.Link
	for _, link = range c.statusDoc.Links {
		if link.Rel == linkRel {
			break
		}
	}
	if link.Href == "" {
		log.Errorf("The %s link is missing", linkRel)
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
			log.Error("Invalid error structure")
		} else {
			log.Warningf("Server message: %s", errResponse.Title)
			err = errors.New("Server error " + strconv.Itoa(r.StatusCode))
		}
		return err
	}

	// get the new status document
	newStatusDoc := new(lic.StatusDoc)
	err := json.NewDecoder(r.Body).Decode(newStatusDoc)
	if err != nil {
		log.Error("Invalid updated status doc structure")
		return err
	}

	// check that the timestamp of the status document has been updated
	if !newStatusDoc.Updated.Status.After(*c.statusDoc.Updated.Status) {
		log.Errorf("The status doc timestamp has not been updated")
	}
	// test if an event has been added to the status document
	if len(newStatusDoc.Events) != len(c.statusDoc.Events)+1 {
		log.Errorf("A new event should have been created")
	} else {
		lastIndex := len(newStatusDoc.Events) - 1
		log.Infof("The last event is of type %s", newStatusDoc.Events[lastIndex].Type)
	}

	// update the current status doc
	c.statusDoc = newStatusDoc

	return nil
}

// setLinkUrl replaces the param of a templated URL by test params
func setLinkUrl(templatedUrl string) (url string, err error) {

	rx, err := regexp.Compile(`\{\?.*\}`)
	if err != nil {
		return
	}
	// set id and name params
	url = string(rx.ReplaceAll([]byte(templatedUrl), []byte("?id=lcp-checker&name=lcp-checker")))
	return
}
