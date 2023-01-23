// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package check

import (
	"crypto/tls"
	"errors"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

const LCPProfileBasic = "http://readium.org/lcp/basic-profile"

// Perfom tests on the license
func CheckLicense(license License, passphrase string) error {

	// check the profile of the license
	err := checkLicenseProfile(license)
	if err != nil {
		return err
	}
	log.Info("Profile ", strings.Split(license.Encryption.Profile, "http://readium.org/lcp/")[1])

	// display the date of issue of the license
	log.Info("Issued on ", license.Issued.Format(time.RFC822))

	// check the certificate chain
	err = checkCertificateChain(license)
	if err != nil {
		return err
	}

	// check that the issue of the license predates the expiration of the provider certificate
	// note: there is no issue if the creation of a certificate comes after the last update of a license

	// check the signature of the license
	var certificate tls.Certificate
	err = checkSignature(license, certificate)
	if err != nil {
		return err
	}

	// check the mime-type of the link to the publication

	// check that the publication can be fetched

	// check the mime-type of the link to the status document

	// check that the status doc is accessed via https

	// check the mime-type of the hint page

	// check that the hint page can be fetched

	// check the format of the content key (64 bytes after base64 decoding)

	// display the text hint and the passphrase passed as parameter
	if passphrase == "" {
		log.Info("No passphrase was passed as a parameter")
	}

	// check the value of the key_check property
	if passphrase != "" {
		return nil // todo
	}

	// check license rights
	// if both are present, start must be before end

	return nil
}

// Verifies the profile of the license
func checkLicenseProfile(license License) error {
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
func checkCertificateChain(license License) error {
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

// Verifies the signature of the license
func checkSignature(license License, cert tls.Certificate) error {

	// remove the current signature from the license

	// verify the signature of the license

	return nil
}
