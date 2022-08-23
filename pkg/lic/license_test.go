// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"crypto/rand"
	"crypto/tls"
	"testing"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/google/uuid"
	"syreclabs.com/go/faker"
)

func setConfig() *conf.Config {

	c := conf.Config{
		Certificate: conf.Certificate{
			Cert:       "../test/cert/cert-edrlab-test.pem",
			PrivateKey: "../test/cert/privkey-edrlab-test.pem",
		},
		License: conf.License{
			Provider: "http://edrlab.org",
			Profile:  "http://readium.org/lcp/basic-profile",
			Links: map[string]string{
				"status": "http://localhost/status/{license_id}",
				"hint":   "https://www.edrlab.org/lcp-help/{license_id}",
			},
		},
	}

	return &c
}

func TestLicense(t *testing.T) {

	conf := setConfig()

	// cert
	cert, err := tls.LoadX509KeyPair(conf.Certificate.Cert, conf.Certificate.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

	// publication
	pub := stor.PublicationInfo{}
	pub.UUID = uuid.New().String()
	pub.Title = faker.Company().CatchPhrase()
	pub.EncryptionKey = make([]byte, 16)
	rand.Read(pub.EncryptionKey)
	pub.Location = faker.Internet().Url()
	pub.ContentType = "application/epub+zip"
	pub.Size = uint32(faker.Number().NumberInt(5))
	pub.Checksum = faker.Lorem().Characters(16)

	// license info
	start := time.Now()
	end := start.AddDate(0, 0, 10)

	licInfo := stor.LicenseInfo{}
	licInfo.UUID = uuid.New().String()
	licInfo.Provider = "https://edrlab.org"
	licInfo.CreatedAt = start
	licInfo.Start = &start
	licInfo.End = &end
	licInfo.Print = int32(-1)
	licInfo.Copy = int32(-1)

	// user info
	userInfo := UserInfo{
		ID:        uuid.New().String(),
		Email:     faker.Internet().Email(),
		Name:      faker.Name().Name(),
		Encrypted: []string{"email", "name"},
	}

	// encryption info
	encryption := Encryption{
		Profile: LCP_Basic_Profile,
		UserKey: UserKey{
			TextHint: "A textual hint for your passphrase.",
		},
	}

	passhash := "FAEB00CA518BEA7CB11A7EF31FB6183B489B1B6EADB792BEC64A03B3F6FF80A8"

	license, err := NewLicense(conf.License, &cert, &pub, &licInfo, &userInfo, &encryption, passhash)

	if err != nil {
		t.Log(err)
		t.Fatal("failed to generate a license.")
	}

	if license.UUID == "" {
		t.Fatal("failed to get the license uuid.")
	}

	/*
		// visual clue
		b, err := json.MarshalIndent(license, "", "\t")
		if err != nil {
			t.Log(err)
			t.Fatal("failed to marshal a license.")
		}

		fmt.Println(string(b))
	*/

}
