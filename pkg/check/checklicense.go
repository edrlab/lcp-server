// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
)

const LCPProfileBasic = "http://readium.org/lcp/basic-profile"

// CheckLicense perfoms multiple tests on a license
func CheckLicense(license lic.License, passphrase string) error {

	// check that the provider is a URL
	parsedURL, err := url.Parse(license.Provider)
	if err != nil {
		return errors.New("error parsing the provider url")
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return errors.New("the provider id must be an http or https url")
	}

	// check that the license id is not empty
	if license.UUID == "" {
		return errors.New("the id of the license is empty")
	}

	// display uuid and the date of issue of the license
	log.Info("License id ", license.UUID)
	log.Info("Issued on ", license.Issued.Format(time.RFC822))

	// check the profile of the license
	err = checkLicenseProfile(license)
	if err != nil {
		return err
	}
	log.Info("Profile ", strings.Split(license.Encryption.Profile, "http://readium.org/lcp/")[1])

	// check the format of the content key (64 bytes after base64 decoding)

	// check the certificate chain
	err = checkCertificateChain(license)
	if err != nil {
		return err
	}

	// check the date of last update of the license
	var endCertificate time.Time
	// TODO: get the real end date of the certificate
	endCertificate = time.Date(2030, time.January, 1, 0, 0, 0, 0, time.UTC)
	err = checkLastUpdate(license, endCertificate)
	if err != nil {
		return err
	}

	if len(license.Links) == 0 {
		return errors.New("this license contains no links")
	}

	// check access to the publication link
	err = checkPublicationLink(license)
	if err != nil {
		return err
	}

	// check access to the status document
	err = checkStatusDocLink(license)
	if err != nil {
		return err
	}

	// check access to the hint page
	err = checkHintPageLink(license)
	if err != nil {
		return err
	}

	// check user info
	err = checkUserInfo(license)
	if err != nil {
		return err
	}

	// check license rights
	err = checkLicenseRights(license)
	if err != nil {
		return err
	}

	// check the signature of the license
	var certificate tls.Certificate
	err = checkSignature(license, certificate)
	if err != nil {
		return err
	}

	// display the text hint and the passphrase passed as parameter
	if passphrase == "" {
		log.Info("No passphrase was passed as a parameter")
	}

	// check the value of the key_check property
	if passphrase != "" {
		return nil // todo
	}

	return nil
}

// Verifies the profile of the license
func checkLicenseProfile(license lic.License) error {
	match, err := regexp.MatchString(
		"http://readium.org/lcp/(basic-profile|1.0|2.[0-9x])",
		license.Encryption.Profile)
	if err != nil {
		return err
	}
	if !match {
		return errors.New("incorrect profile value")
	}
	return nil
}

// Verifies the certificate chain
func checkCertificateChain(license lic.License) error {
	/*
		var cacert []byte
		var err error
		if license.Encryption.Profile == LCPProfileBasic {
			cacert, err = cact.ReadFile("data/cacert-edrlab-test.pem")
		} else {
			cacert, err = cacp.ReadFile("data/cacert-edrlab-prod.pem")
		}
		if err != nil {
			return err
		}
	*/

	return nil
}

// Verifies that the date of last update predates the expiration of the provider certificate
// note: there is no issue if the creation of a certificate happens after the last update of a license;
// this happens when the certificate is updated.
func checkLastUpdate(license lic.License, endCertificate time.Time) error {

	var lastUpdated time.Time
	if license.Updated == nil {
		lastUpdated = license.Issued
	} else {
		lastUpdated = *license.Updated
		// verifies that the date of update is after the date of issue
		if lastUpdated.Before(license.Issued) {
			return fmt.Errorf("incorrect date of update %s, should be after the date of issue %s", lastUpdated.String(), license.Issued.String())
		}
	}
	if lastUpdated.After(endCertificate) {
		return fmt.Errorf("incorrect date of last update %s, should be before the date of expiration of the certificate %s", lastUpdated.String(), endCertificate.String())
	}
	return nil
}

// Verifies the signature
func checkSignature(license lic.License, cert tls.Certificate) error {

	// remove the current signature from the license

	// verify the signature of the license

	return nil
}

// Verifies the publication link
func checkPublicationLink(license lic.License) error {

	var pubType, pubHref string
	for _, l := range license.Links {
		if l.Rel == "publication" {
			pubType = l.Type
			pubHref = l.Href
		}
	}
	if pubHref == "" {
		return errors.New("this license does not contain a link to a an encrypted publication")
	}

	// check the mime-type of the link to the publication
	mimetypes := [4]string{
		"application/epub+zip",
		"application/pdf+lcp",
		"application/audiobook+lcp",
		"application/divina+lcp",
	}
	var found bool
	for _, v := range mimetypes {
		if v == pubType {
			found = true
		}
	}
	if !found {
		return fmt.Errorf("unknown publication mime type %s", pubType)
	}

	// check that the publication can be fetched
	resp, err := http.Head(pubHref)
	if err != nil {
		return errors.New("unreachable publication URL")
	}
	resp.Body.Close()

	return nil
}

// Verifies the publication link
func checkStatusDocLink(license lic.License) error {

	var sdType, sdHref string
	for _, l := range license.Links {
		if l.Rel == "status" {
			sdType = l.Type
			sdHref = l.Href
		}
	}
	if sdHref == "" {
		return errors.New("this license does not contain a link to a status document")
	}
	if sdType != "application/vnd.readium.license.status.v1.0+json" {
		return fmt.Errorf("invalid status document mime type %s", sdType)
	}
	// check that the status document URL is based on https
	parsedURL, err := url.Parse(sdHref)
	if err != nil {
		return fmt.Errorf("error parsing the status document url : %w", err)
	}
	if parsedURL.Scheme != "https" {
		log.Warning("the link to status document should be an https url")
	}

	// check that the status document can be fetched
	resp, err := http.Head(sdHref)
	if err != nil {
		return errors.New("unreachable status document URL")
	}
	resp.Body.Close()

	return nil
}

// Verifies the hint page link
func checkHintPageLink(license lic.License) error {

	var hintType, hintHref string
	for _, l := range license.Links {
		if l.Rel == "hint" {
			hintType = l.Type
			hintHref = l.Href
		}
	}
	if hintHref == "" {
		return errors.New("this license does not contain a link to a an hint page")
	}

	if hintType != "text/html" {
		return fmt.Errorf("invalid hint page mime type %s", hintType)
	}

	// check that the hint page can be fetched
	resp, err := http.Head(hintHref)
	if err != nil {
		return errors.New("unreachable hint page URL")
	}
	resp.Body.Close()

	return nil
}

// Verifies user info
func checkUserInfo(license lic.License) error {

	// warn if the user id is missing
	if license.User.ID == "" {
		log.Warning("please consider aadding the user id to the license")
	}
	return nil
}

// Verifies license rights
func checkLicenseRights(license lic.License) error {

	// check that the start date is before the end date (if any)
	if license.Rights.Start != nil && license.Rights.End != nil {
		if license.Rights.Start.After(*license.Rights.End) {
			return fmt.Errorf(("invalid rights: start is after end"))
		}
	}

	// warn if the copy and print rights are low
	if license.Rights.Copy != nil {
		if *license.Rights.Copy < 1000 {
			log.Warning("please consider allowing more than 1000 copied characters")
		}
	}
	if license.Rights.Print != nil {
		if *license.Rights.Print < 10 {
			log.Warning("please consider allowing more than 10 printed pages")
		}
	}

	return nil
}
