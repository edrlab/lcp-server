// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"log"
	"reflect"
	"strings"

	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/readium/readium-lcp-server/crypto"
	"github.com/readium/readium-lcp-server/sign"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

const (
	ContentType_LCP_JSON         = "application/vnd.readium.lcp.license.v1.0+json"
	ContentType_LSD_JSON         = "application/vnd.readium.license.status.v1.0+json"
	ContentType_TEXT_HTML        = "text/html"
	ContentType_JSON             = "application/json"
	ContentType_FORM_URL_ENCODED = "application/x-www-form-urlencoded"
)

const (
	LCP_Basic_Profile = "http://readium.org/lcp/basic-profile"
	LCP_10_Profile    = "http://readium.org/lcp/profile-1.0"
)

// ====
// LCP License
// ====

type License struct {
	Provider   string         `json:"provider"`
	UUID       string         `json:"id"`
	Issued     time.Time      `json:"issued"`
	Updated    *time.Time     `json:"updated,omitempty"`
	Encryption Encryption     `json:"encryption"`
	Links      *[]Link        `json:"links,omitempty"` // TODO : see if a pointer is needed here
	User       UserInfo       `json:"user"`
	Rights     UserRights     `json:"rights"`
	Signature  sign.Signature `json:"signature"`
}

type Encryption struct { // Used for license generation
	Profile    string     `json:"profile,omitempty"`
	ContentKey ContentKey `json:"content_key,omitempty"` // Not used for license generation
	UserKey    UserKey    `json:"user_key"`
}

type Link struct {
	Rel       string `json:"rel"`
	Href      string `json:"href"`
	Type      string `json:"type,omitempty"`
	Title     string `json:"title,omitempty"`
	Profile   string `json:"profile,omitempty"`
	Templated bool   `json:"templated,omitempty"`
	Size      int64  `json:"length,omitempty"`
	Checksum  string `json:"hash,omitempty"`
}

type UserInfo struct { // Used for license generation
	ID        string   `json:"id"`
	Email     string   `json:"email,omitempty"`
	Name      string   `json:"name,omitempty"`
	Encrypted []string `json:"encrypted,omitempty"`
}

type UserRights struct { // Used for license generation
	Start *time.Time `json:"start,omitempty"`
	End   *time.Time `json:"end,omitempty"`
	Print *int32     `json:"print,omitempty"`
	Copy  *int32     `json:"copy,omitempty"`
}

type ContentKey struct {
	Algorithm string `json:"algorithm,omitempty"`
	Value     []byte `json:"encrypted_value,omitempty"`
}

type UserKey struct {
	Algorithm string `json:"algorithm,omitempty"` // Not used for license generation
	TextHint  string `json:"text_hint,omitempty"`
	Keycheck  []byte `json:"key_check,omitempty"` // Not used for license generation
}

// ====

const SHA256_URI string = "http://www.w3.org/2001/04/xmlenc#sha256"

// NewLicense generates a license from db info, request data and config data
func NewLicense(conf conf.License, cert *tls.Certificate, pubInfo *stor.PublicationInfo, licInfo *stor.LicenseInfo, userInfo *UserInfo, encryption *Encryption, passhash string) (*License, error) {

	l := &License{
		UUID:     licInfo.UUID,
		Provider: licInfo.Provider,
		Issued:   licInfo.CreatedAt,
	}

	log.Printf("License %s generated on %s", l.UUID, l.Issued.Format(time.RFC822))

	userKey, err := setEncryption(conf, l, pubInfo, encryption, passhash)
	if err != nil {
		return nil, err
	}

	// links
	setLinks(conf, l, pubInfo)

	// user
	err = setUser(conf, l, userInfo, userKey)
	if err != nil {
		return nil, err
	}

	//rights
	err = setRights(conf, l, licInfo)
	if err != nil {
		return nil, err
	}

	// signature
	err = setSignature(conf, l, cert)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// setEncryption sets the encryption structure in the license
// returns the user key, which will be used later to encrypt user info
func setEncryption(conf conf.License, l *License, pub *stor.PublicationInfo, encryption *Encryption, passhash string) ([]byte, error) {

	if encryption.Profile == "" {
		encryption.Profile = conf.Profile // by default from the config
	}
	if encryption.Profile == "" {
		return nil, errors.New("failed to set the license profile")
	}

	// generate the user key
	userKey, err := GenerateUserKey(encryption.Profile, passhash)
	if err != nil {
		return nil, err
	}

	// encrypt the content key with the user key
	contentKeyEncrypter := crypto.NewAESEncrypter_CONTENT_KEY()
	encryption.ContentKey.Algorithm = contentKeyEncrypter.Signature()
	encryption.ContentKey.Value = encryptKey(contentKeyEncrypter, pub.EncryptionKey, userKey[:])

	// build the key check
	encryption.UserKey.Algorithm = SHA256_URI
	userKeyCheckEncrypter := crypto.NewAESEncrypter_USER_KEY_CHECK()
	encryption.UserKey.Keycheck, err = buildKeyCheck(l.UUID, userKeyCheckEncrypter, userKey[:])
	if err != nil {
		return nil, err
	}

	l.Encryption = *encryption
	return userKey, nil

}

func encryptKey(encrypter crypto.Encrypter, key []byte, kek []byte) []byte {
	var out bytes.Buffer
	in := bytes.NewReader(key)
	encrypter.Encrypt(kek[:], in, &out)
	return out.Bytes()
}

func buildKeyCheck(licenseID string, encrypter crypto.Encrypter, key []byte) ([]byte, error) {

	var out bytes.Buffer
	err := encrypter.Encrypt(key, bytes.NewBufferString(licenseID), &out)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// setLinks sets the links structure in the license
func setLinks(conf conf.License, l *License, pub *stor.PublicationInfo) {

	var links []Link

	// set the links found in the config (status, hint)
	confLinks := conf.Links
	for key := range confLinks {
		link := Link{Href: confLinks[key], Rel: key}
		links = append(links, link)
	}

	// customize the status and hint links
	for i := 0; i < len(links); i++ {
		// status
		if links[i].Rel == "status" {
			links[i].Href = strings.Replace(links[i].Href, "{license_id}", l.UUID, 1)
			links[i].Type = ContentType_LSD_JSON
		}

		// hint page , which may be associated with a specific license
		if links[i].Rel == "hint" {
			links[i].Href = strings.Replace(links[i].Href, "{license_id}", l.UUID, 1)
			links[i].Type = ContentType_TEXT_HTML
		}
	}

	// set the publication link
	link := Link{
		Rel:      "publication",
		Href:     pub.Location,
		Type:     pub.ContentType,
		Title:    pub.Title,
		Size:     int64(pub.Size),
		Checksum: pub.Checksum,
	}
	links = append(links, link)

	l.Links = &links
}

// setUser sets the user structure in the license
func setUser(conf conf.License, l *License, userInfo *UserInfo, userKey []byte) error {

	// encrypt user info fields
	fieldsEncrypter := crypto.NewAESEncrypter_FIELDS()
	err := encryptFields(fieldsEncrypter, userInfo, userKey[:])
	if err != nil {
		return err
	}
	// set user info in the license
	l.User = *userInfo
	return nil
}

func encryptFields(encrypter crypto.Encrypter, userInfo *UserInfo, key []byte) error {
	for _, toEncrypt := range userInfo.Encrypted {
		var out bytes.Buffer
		field := getField(userInfo, toEncrypt)
		err := encrypter.Encrypt(key[:], bytes.NewBufferString(field.String()), &out)
		if err != nil {
			return err
		}
		field.Set(reflect.ValueOf(base64.StdEncoding.EncodeToString(out.Bytes())))
	}
	return nil
}

func getField(u *UserInfo, field string) reflect.Value {
	v := reflect.ValueOf(u).Elem()
	c := cases.Title(language.Und, cases.NoLower)
	return v.FieldByName(c.String(field))
}

// setRights sets the rights structure in the license
func setRights(conf conf.License, l *License, licInfo *stor.LicenseInfo) error {

	l.Rights.Start = licInfo.Start
	l.Rights.End = licInfo.End
	if licInfo.Print != -1 {
		l.Rights.Print = &licInfo.Print
	}
	if licInfo.Copy != -1 {
		l.Rights.Copy = &licInfo.Copy
	}
	return nil
}

// setSignature sets the signature of the license
func setSignature(conf conf.License, l *License, cert *tls.Certificate) error {

	if cert == nil {
		return errors.New("failed to sign the license, cert not set")
	}
	sig, err := sign.NewSigner(cert)
	if err != nil {
		return err
	}
	res, err := sig.Sign(l)
	if err != nil {
		return err
	}
	l.Signature = res

	return nil
}
