// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"bytes"
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/hex"
	"errors"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/edrlab/lcp-server/pkg/crypto"
	"github.com/edrlab/lcp-server/pkg/lic"
	log "github.com/sirupsen/logrus"
)

const LCPProfileBasic = "http://readium.org/lcp/basic-profile"

//go:embed data/cacert-edrlab-test.pem data/cacert-edrlab-prod.pem
var cafs embed.FS

// CheckLicense perfoms multiple tests on a license
func (c *LicenseChecker) CheckLicense(passphrase string) error {

	// check that the provider is a URL
	parsedURL, err := url.Parse(c.license.Provider)
	if err != nil {
		log.Error("The provider of a license must be expressed as a url")
	} else if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		log.Error("The provider id must be an http or https url")
	}

	// check that the license id is not empty
	if c.license.UUID == "" {
		log.Error("A license must have an identifier")
	}

	// check the profile of the license
	err = c.CheckLicenseProfile()
	if err != nil {
		return err
	}

	// check the format of the content key
	err = c.CheckContentKeyFormat()
	if err != nil {
		return err
	}

	// check the certificate chain
	notAfter, err := c.CheckCertificateChain()
	if err != nil {
		return err
	}

	// check the date of issue of the license
	err = c.CheckIssueDate(notAfter)
	if err != nil {
		return err
	}

	if len(c.license.Links) == 0 {
		log.Error("A license must have links")
	}

	// check access to the publication link
	err = c.CheckPublicationLink()
	if err != nil {
		return err
	}

	// check access to the status document
	err = c.CheckStatusDocLink()
	if err != nil {
		return err
	}

	// check access to the hint page
	err = c.CheckHintPageLink()
	if err != nil {
		return err
	}

	// check user info
	err = c.CheckUserInfo()
	if err != nil {
		return err
	}

	// check license rights
	err = c.CheckLicenseRights()
	if err != nil {
		return err
	}

	// check the signature of the license
	err = c.CheckSignature()
	if err != nil {
		return err
	}

	// check the value of the key_check property
	if passphrase != "" {
		err = c.CheckPassphrase(passphrase)
		if err != nil {
			return err
		}
	} else {
		log.Info("As no passphrase is provided, the key_check property is not checked")
	}

	return nil
}

// Verifies the profile of the license
func (c *LicenseChecker) CheckLicenseProfile() error {
	if c.license.Encryption.Profile == "" {
		log.Error("Empty profile")
		return nil
	}

	match, err := regexp.MatchString(
		"http://readium.org/lcp/(basic-profile|profile-1.0|profile-2.[0-9x])",
		c.license.Encryption.Profile)
	if err != nil {
		return err
	}
	if !match {
		log.Errorf("The profile value %s is incorrect", c.license.Encryption.Profile)
	}
	return nil
}

// Verifies the format of the content key
func (c *LicenseChecker) CheckContentKeyFormat() error {

	// the content key must be 64 bytes long (after base64 decoding)
	if len(c.license.Encryption.ContentKey.Value) != 64 {
		log.Errorf("Invalid content key length: %d", len(c.license.Encryption.ContentKey.Value))
		return errors.New("invalid content key length")
	}

	return nil
}

// Verifies the certificate chain
func (c *LicenseChecker) CheckCertificateChain() (*time.Time, error) {

	var cacert []byte
	var err error
	if c.license.Encryption.Profile == LCPProfileBasic {
		cacert, err = cafs.ReadFile("data/cacert-edrlab-test.pem")
	} else {
		cacert, err = cafs.ReadFile("data/cacert-edrlab-prod.pem")
	}
	if err != nil {
		return nil, err
	}

	roots := x509.NewCertPool()
	ok := roots.AppendCertsFromPEM(cacert)
	if !ok {
		return nil, errors.New("failed to parse root certificate")
	}

	if c.license.Signature.Certificate == nil {
		return nil, errors.New("failed to check certificate chain: no signature present")
	}
	// parse the provider certificate (as ASN.1 DER data)
	cert, err := x509.ParseCertificate(c.license.Signature.Certificate)
	if err != nil {
		return nil, errors.New("failed to parse the certificate:" + err.Error())
	}

	opts := x509.VerifyOptions{
		Roots: roots,
	}

	// Verify the certificate validity
	// An expired certificate is not an issue
	if _, err := cert.Verify(opts); err != nil && cert.NotAfter.After(time.Now()) {
		log.Warning("It appears that a test certificate is used but a production profile declared, or vice versa.")
		return nil, errors.New("failed to verify the certificate: " + err.Error())
	}

	if cert.NotAfter.Before(time.Now()) {
		log.Infof("This certificate has expired on %s", cert.NotAfter.Format(time.RFC822))
	}

	return &cert.NotAfter, nil
}

// Verifies that the license was issued before the expiration of the provider certificate.
// Note: there is no issue if the license was issued before the certificate was created:
// this happens when a new provider certificate is installed and a fresh license is generated.
func (c *LicenseChecker) CheckIssueDate(endCertificate *time.Time) error {

	if endCertificate == nil {
		return errors.New("cannot check last license update with a nil certificate end date")
	}

	if c.license.Issued.After(*endCertificate) {
		log.Errorf("Incorrect license issue date %s, the certificate had already expired on %s", c.license.Issued.Format(time.RFC822), endCertificate.Format(time.RFC822))
	}
	return nil
}

// Verifies the signature
func (c *LicenseChecker) CheckSignature() error {

	err := c.license.CheckSignature()
	if err != nil {
		log.Errorf("The signature of the license is incorrect")
	}
	return nil
}

// Verifies the publication link
func (c *LicenseChecker) CheckPublicationLink() error {

	var pubType, pubHref string
	for _, l := range c.license.Links {
		if l.Rel == "publication" {
			pubType = l.Type
			pubHref = l.Href
		}
	}
	if pubHref == "" {
		log.Error("A license must link to an encrypted publication")
		return nil
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
	err := CheckResource(pubHref)
	if err != nil {
		log.Errorf("The publication at %s is unreachable", pubHref)
	}
	return nil
}

// Verifies the status document link
func (c *LicenseChecker) CheckStatusDocLink() error {

	var sdType, sdHref string
	for _, l := range c.license.Links {
		if l.Rel == "status" {
			sdType = l.Type
			sdHref = l.Href
		}
	}
	if sdHref == "" {
		log.Error("A license must link to a status document")
		return nil
	}
	if sdType != "application/vnd.readium.license.status.v1.0+json" {
		log.Errorf("The mime type of the status document (%s) is invalid", sdType)
	}
	// check that the status document URL is based on https
	parsedURL, err := url.Parse(sdHref)
	if err != nil {
		log.Errorf("The status document url could not be parsed: %v", err)
	} else {
		if parsedURL.Scheme != "https" {
			log.Warning("The link to status document should be an https url")
		}
	}

	// check that the status document can be fetched
	err = CheckResource(sdHref)
	if err != nil {
		log.Errorf("The status document at %s is unreachable", sdHref)
	}
	return nil
}

// Verifies the hint page link
func (c *LicenseChecker) CheckHintPageLink() error {

	var hintType, hintHref string
	for _, l := range c.license.Links {
		if l.Rel == "hint" {
			hintType = l.Type
			hintHref = l.Href
		}
	}
	if hintHref == "" {
		log.Error("A license must link to an hint page")
		return nil
	}

	if hintType != "text/html" {
		log.Errorf("The mime type of the hint page (%s) is invalid", hintType)
	}

	// check that the hint page can be fetched
	err := CheckResource(hintHref)
	if err != nil {
		log.Errorf("The hint page at %s unreachable", hintHref)
	}
	return nil
}

// Verifies user info
func (c *LicenseChecker) CheckUserInfo() error {

	// warn if the user id is missing
	if c.license.User.ID == "" {
		log.Warning("Please consider adding a user id to the license")
	}
	return nil
}

// Verifies license rights
func (c *LicenseChecker) CheckLicenseRights() error {

	var tstart, tend time.Time
	var start, end, copy, print string
	und := "undefined"
	if c.license.Rights.Start != nil {
		tstart = *c.license.Rights.Start
		start = tstart.Format(time.RFC822)
	} else {
		start = und
	}
	if c.license.Rights.End != nil {
		tend = *c.license.Rights.End
		end = tend.Format(time.RFC822)
	} else {
		end = und
	}
	if c.license.Rights.Copy != nil {
		copy = strconv.Itoa(int(*c.license.Rights.Copy))
	} else {
		copy = und
	}
	if c.license.Rights.Print != nil {
		print = strconv.Itoa(int(*c.license.Rights.Print))
	} else {
		print = und
	}
	log.Infof("Rights: Start %s, End %s, Copy %s, Print %s", start, end, copy, print)

	// check that the start date is before the end date (if any)
	if c.license.Rights.Start != nil && c.license.Rights.End != nil {
		if c.license.Rights.Start.After(*c.license.Rights.End) {
			log.Error(("Invalid rights: start is after end"))
		}
	}

	// advise if the end date is in the past or the start date in the future
	if c.license.Rights.Start != nil {
		if c.license.Rights.Start.After(time.Now()) {
			log.Info(("The start date is in the future: it is not usable yet"))
		}
	}
	if c.license.Rights.End != nil && c.license.Rights.End.Before(time.Now()) {
		log.Info(("The end date is in the past: this license has expired"))
	}

	// if the status doc has already been fetched
	// check that the end date of the license is compatible with the status of the license
	if c.statusDoc != nil {
		if c.statusDoc.Status == "revoked" || c.statusDoc.Status == "returned" || c.statusDoc.Status == "cancelled" || c.statusDoc.Status == "expired" {
			if c.license.Rights.End != nil && c.license.Rights.End.After(time.Now()) {
				log.Errorf("The license is in %s state, its end date should be passed", c.statusDoc.Status)
			}
		}
	}

	// warn if the copy and print rights are low
	if c.license.Rights.Copy != nil {
		if *c.license.Rights.Copy < 5000 {
			log.Warning("Please consider allowing at least 5000 characters to be copied")
		}
	}
	if c.license.Rights.Print != nil {
		if *c.license.Rights.Print < 10 {
			log.Warning("Please consider allowing at least 10 pages to be printed")
		}
	}
	return nil
}

// Check the passphrase
func (c *LicenseChecker) CheckPassphrase(passphrase string) error {

	keycheck := c.license.Encryption.UserKey.Keycheck

	//fmt.Println("keycheck:", base64.StdEncoding.EncodeToString(keycheck))

	if len(keycheck) != 64 {
		log.Errorf("Key_check is %d bytes long, should be 64", len(keycheck))
		return nil
	}

	// calculate the hash of the passphrase, hex encode it
	hash := sha256.Sum256([]byte(passphrase))
	passhash := hex.EncodeToString(hash[:])

	//fmt.Println("passhash: ", passhash)

	// regenerate the user key
	userKey, err := lic.GenerateUserKey(c.license.Encryption.Profile, passhash)
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
	if result.String() != c.license.UUID {
		log.Error("The passphrase passed as parameter seems incorrect (key check failed)")
	} else {
		log.Info("The passphrase passed as parameter is correct (key check passed)")
	}
	return nil
}

// ShowLicenseInfo displays information about the license
func (c *LicenseChecker) ShowLicenseInfo() error {

	// display uuid and the date of issue of the license
	log.Info("License id ", c.license.UUID)
	log.Info("Issued on ", c.license.Issued.Format(time.RFC822))
	log.Info("Using ", strings.Split(c.license.Encryption.Profile, "http://readium.org/lcp/")[1])

	return nil
}
