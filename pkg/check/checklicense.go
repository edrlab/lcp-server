// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/edrlab/lcp-server/pkg/lic"
	"github.com/readium/readium-lcp-server/crypto"
	log "github.com/sirupsen/logrus"
)

const LCPProfileBasic = "http://readium.org/lcp/basic-profile"

// CheckLicense perfoms multiple tests on a license
func CheckLicense(license lic.License, passphrase string) error {

	// check that the provider is a URL
	parsedURL, err := url.Parse(license.Provider)
	if err != nil {
		log.Error("The provider of a license must be expressed as a url")
	}
	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		log.Error("The provider id must be an http or https url")
	}

	// check that the license id is not empty
	if license.UUID == "" {
		log.Error("A license must have an identifier")
	}

	// display uuid and the date of issue of the license
	log.Info("License id ", license.UUID)
	log.Info("Issued on ", license.Issued.Format(time.RFC822))

	// check the profile of the license
	err = checkLicenseProfile(license)
	if err != nil {
		return err
	}

	// check the format of the content key (64 bytes after base64 decoding)
	// TODO

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
		log.Error("A license must have links")
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
	err = checkSignature(license)
	if err != nil {
		return err
	}

	// check the value of the key_check property
	if passphrase != "" {
		err = checkPassphrase(license, passphrase)
		if err != nil {
			return err
		}
	} else {
		log.Info("As no passphrase is provided, the key_check property is not checked")
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
		log.Errorf("The profile value %s is incorrect", license.Encryption.Profile)
	}
	log.Info("Profile ", strings.Split(license.Encryption.Profile, "http://readium.org/lcp/")[1])
	return nil
}

// Verifies the certificate chain
func checkCertificateChain(license lic.License) error {

	// TODO
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
			log.Errorf("Incorrect date of update %s, should be after the date of issue %s", lastUpdated.String(), license.Issued.String())
		}
	}
	if lastUpdated.After(endCertificate) {
		log.Errorf("Incorrect date of last update %s, should be before the date of expiration of the certificate %s", lastUpdated.String(), endCertificate.String())
	}
	return nil
}

// Verifies the signature
func checkSignature(license lic.License) error {

	err := license.CheckSignature()
	if err != nil {
		log.Errorf("The signature of the license is incorrect: %v", err)
	}
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
		log.Error("A license must link to an encrypted publication")
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
		log.Errorf("The mime type of the publication (%s) is unsupported", pubType)
	}

	// check that the publication can be fetched
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	_, err := client.Head(pubHref)
	if err != nil {
		log.Errorf("The publication at %s is unreachable", pubHref)
	}
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
		log.Error("A license must link to a status document")
	}
	if sdType != "application/vnd.readium.license.status.v1.0+json" {
		log.Errorf("The mime type of the status document (%s) is invalid", sdType)
	}
	// check that the status document URL is based on https
	parsedURL, err := url.Parse(sdHref)
	if err != nil {
		log.Errorf("The status document url could not be parsed: %w", err)
	} else {
		if parsedURL.Scheme != "https" {
			log.Warning("The link to status document should be an https url")
		}
	}

	// check that the status document can be fetched
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	_, err = client.Head(sdHref)
	if err != nil {
		log.Errorf("The status document at %s is unreachable", sdHref)
	}
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
		log.Error("A license must link to an hint page")
	}

	if hintType != "text/html" {
		log.Errorf("The mime type of the hint page (%s) is invalid", hintType)
	}

	// check that the hint page can be fetched
	client := http.Client{
		Timeout: 1 * time.Second,
	}
	_, err := client.Head(hintHref)
	if err != nil {
		log.Errorf("The hint page at %s unreachable", hintHref)
	}
	return nil
}

// Verifies user info
func checkUserInfo(license lic.License) error {

	// warn if the user id is missing
	if license.User.ID == "" {
		log.Warning("Please consider adding a user id to the license")
	}
	return nil
}

// Verifies license rights
func checkLicenseRights(license lic.License) error {

	// check that the start date is before the end date (if any)
	if license.Rights.Start != nil && license.Rights.End != nil {
		if license.Rights.Start.After(*license.Rights.End) {
			log.Error(("Invalid rights: start is after end"))
		}
	}

	// warn if the copy and print rights are low
	if license.Rights.Copy != nil {
		if *license.Rights.Copy < 5000 {
			log.Warning("Please consider allowing at least 5000 characters to be copied")
		}
	}
	if license.Rights.Print != nil {
		if *license.Rights.Print < 10 {
			log.Warning("Please consider allowing at least 10 pages to be printed")
		}
	}
	return nil
}

// Check the passphrase
func checkPassphrase(license lic.License, passphrase string) error {

	keycheck := license.Encryption.UserKey.Keycheck

	//fmt.Println("keycheck:", base64.StdEncoding.EncodeToString(keycheck))

	if len(keycheck) != 64 {
		log.Errorf("Key_check is %d bytes long, should be 64", len(keycheck))
		return nil
	}

	// calculate the hash of the passphrase, hex encore it
	hash := sha256.Sum256([]byte(passphrase))
	passhash := hex.EncodeToString(hash[:])

	//fmt.Println("passhash: ", passhash)

	// regenerate the user key
	userKey, err := lic.GenerateUserKey(license.Encryption.Profile, passhash)
	if err != nil {
		return err
	}

	// decrypt the key check using the user key
	encrypter := crypto.NewAESEncrypter_USER_KEY_CHECK()
	decrypter, ok := encrypter.(crypto.Decrypter)
	if !ok {
		return errors.New("failed to create a decrypter")
	}
	var result bytes.Buffer
	err = decrypter.Decrypt(crypto.ContentKey(userKey), bytes.NewBuffer(keycheck), &result)
	if err != nil {
		return err
	}

	// check that the decrypted key check is the license id
	if result.String() != license.UUID {
		log.Errorf("The passphrase passed as parameter seems incorrect (key check failed)")
	}
	return nil
}
