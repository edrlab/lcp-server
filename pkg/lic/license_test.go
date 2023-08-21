// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"crypto/rand"
	"crypto/tls"
	"os"
	"testing"
	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/google/uuid"
	"syreclabs.com/go/faker"
)

// some global vars shares by all tests
var LicCt LicenseCtrl
var Pub stor.Publication
var LicInfo stor.LicenseInfo

func setConfig() *conf.Config {

	c := conf.Config{
		PublicBaseUrl: "http://localhost:8081",
		Certificate: conf.Certificate{
			Cert:       "../test/cert/cert-edrlab-test.pem",
			PrivateKey: "../test/cert/privkey-edrlab-test.pem",
		},
		License: conf.License{
			Provider: "http://edrlab.org",
			Profile:  "http://readium.org/lcp/basic-profile",
			HintLink: "https://www.edrlab.org/lcp-help/{license_id}",
		},
	}
	return &c
}

func TestMain(m *testing.M) {

	LicCt.Config = setConfig()

	// Create / open an sqlite db in memory
	dsn := "sqlite3://file::memory:?cache=shared"
	LicCt.Store, _ = stor.Init(dsn)

	// create a publication
	Pub.UUID = uuid.New().String()
	Pub.Title = faker.Company().CatchPhrase()
	Pub.EncryptionKey = make([]byte, 16)
	rand.Read(Pub.EncryptionKey)
	Pub.Location = faker.Internet().Url()
	Pub.ContentType = "application/epub+zip"
	Pub.Size = uint32(faker.Number().NumberInt(5))
	Pub.Checksum = faker.Lorem().Characters(16)

	// store the publication in the db
	LicCt.Store.Publication().Create(&Pub)

	// create a license
	start := time.Now()
	end := start.AddDate(0, 0, 10)

	LicInfo.UUID = uuid.New().String()
	LicInfo.Provider = "https://edrlab.org"
	LicInfo.CreatedAt = start
	LicInfo.Status = stor.STATUS_READY
	LicInfo.Start = &start
	LicInfo.End = &end
	LicInfo.Print = int32(-1)
	LicInfo.Copy = int32(-1)
	// associate the license with the publication
	LicInfo.PublicationID = Pub.UUID

	// store the license in the db
	LicCt.Store.License().Create(&LicInfo)

	code := m.Run()
	os.Exit(code)
}

func TestLicense(t *testing.T) {

	// cert
	cert, err := tls.LoadX509KeyPair(LicCt.Config.Certificate.Cert, LicCt.Config.Certificate.PrivateKey)
	if err != nil {
		t.Fatal(err)
	}

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

	license, err := NewLicense(LicCt.Config, &cert, &Pub, &LicInfo, &userInfo, &encryption, passhash)

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
