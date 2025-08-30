// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package lic

import (
	"bytes"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"errors"
	"math/big"
	"net/url"
	"reflect"
	"strconv"

	log "github.com/sirupsen/logrus"

	"time"

	"github.com/edrlab/lcp-server/pkg/conf"
	"github.com/edrlab/lcp-server/pkg/crypto"
	"github.com/edrlab/lcp-server/pkg/sign"
	"github.com/edrlab/lcp-server/pkg/stor"
	"github.com/jtacoma/uritemplates"
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
// note: a signature is nill when a license is canonicalized for being signed

type License struct {
	Provider   string          `json:"provider"`
	UUID       string          `json:"id"`
	Issued     time.Time       `json:"issued"`
	Updated    *time.Time      `json:"updated,omitempty"`
	Encryption Encryption      `json:"encryption"`
	Links      []Link          `json:"links,omitempty"`
	User       UserInfo        `json:"user"`
	Rights     UserRights      `json:"rights"`
	Signature  *sign.Signature `json:"signature,omitempty"`
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
func NewLicense(config *conf.Config, cert *tls.Certificate, pubInfo *stor.Publication, licInfo *stor.LicenseInfo, userInfo *UserInfo, encryption *Encryption, passhash string) (*License, error) {

	l := &License{
		UUID:     licInfo.UUID,
		Provider: licInfo.Provider,
		Issued:   licInfo.CreatedAt,
		Updated:  licInfo.Updated,
	}

	userKey, err := setEncryption(config.Profile, l, pubInfo, encryption, passhash)
	if err != nil {
		return nil, err
	}

	// links
	setLinks(config.PublicBaseUrl, config.License.HintLink, l, pubInfo)

	// user
	err = setUser(l, userInfo, userKey)
	if err != nil {
		return nil, err
	}

	//rights
	err = setRights(l, licInfo)
	if err != nil {
		return nil, err
	}

	// signature
	err = setSignature(l, cert)
	if err != nil {
		return nil, err
	}

	return l, nil
}

// generateRandomDigit generates a random number between 1 and 9
func generateRandomDigit() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return 0, err
	}
	return int(n.Int64()), nil
}

// setEncryption sets the encryption structure in the license
// returns the user key, which will be used later to encrypt user info
func setEncryption(profile string, l *License, pub *stor.Publication, encryption *Encryption, passhash string) ([]byte, error) {

	if encryption.Profile == "" {
		if profile == "" {
			return nil, errors.New("missing profile value")
		}
		encryption.Profile = profile // by default from the config
	}

	// if the profile contains a jocker, generate a random number between 0 and 9
	if encryption.Profile == "2.x" {
		randomDigit, err := generateRandomDigit()
		if err != nil {
			return nil, errors.New("unable to generate a random digit")
		}
		encryption.Profile = "2." + strconv.Itoa(randomDigit)
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
func setLinks(publicBaseUrl string, hintTemplate string, l *License, pub *stor.Publication) {

	// if the pub location is the specific value below, prefix it with the public base URL
	// and the "resources" route.
	localhost := "http://localhost/"
	if pub.Href != "" && len(pub.Href) > 17 && (pub.Href[:17] == localhost) {
		var err error
		pub.Href, err = url.JoinPath(publicBaseUrl, "resources", pub.Href[17:])
		if err != nil {
			log.Printf("failed to join publication link: %v", err)
		}
	}

	// set the publication link
	pubLink := Link{
		Rel:      "publication",
		Href:     pub.Href,
		Type:     pub.ContentType,
		Title:    pub.Title,
		Size:     int64(pub.Size),
		Checksum: pub.Checksum,
	}
	l.Links = append(l.Links, pubLink)

	// set the status link
	statusHref, err := url.JoinPath(publicBaseUrl, "status", l.UUID)
	if err != nil {
		log.Printf("failed to join status link: %v", err)
		statusHref = publicBaseUrl + "/status/" + l.UUID // fallback
	}
	statusLink := Link{
		Rel:  "status",
		Href: statusHref,
		Type: ContentType_LSD_JSON,
	}
	l.Links = append(l.Links, statusLink)

	// expand the link template for the hint
	template, _ := uritemplates.Parse(hintTemplate)
	values := make(map[string]interface{})
	values["license_id"] = l.UUID
	expanded, err := template.Expand(values)
	if err != nil {
		log.Printf("failed to expand the hint link: %s", template)
		expanded = hintTemplate // fallback
	}

	// set the hint link
	hintLink := Link{
		Rel:  "hint",
		Href: expanded,
		Type: ContentType_TEXT_HTML,
	}
	l.Links = append(l.Links, hintLink)

}

// setUser sets the user structure in the license
func setUser(l *License, userInfo *UserInfo, userKey []byte) error {

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
func setRights(l *License, licInfo *stor.LicenseInfo) error {

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
func setSignature(l *License, cert *tls.Certificate) error {

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
	l.Signature = &res

	return nil
}

// CheckSignature verifies the signature of a license
func (license *License) CheckSignature() error {
	if license.Signature == nil {
		return errors.New("missing signature")
	}

	// extract the signature from the license
	signature := license.Signature
	// raz the embedded signature
	license.Signature = nil

	signChecker, err := sign.NewSignChecker(signature.Certificate, signature.Algorithm)
	if err != nil {
		return err
	}

	err = signChecker.Check(license, signature.Value)
	if err != nil {
		return err
	}
	// put back the signature in place
	license.Signature = signature
	return nil
}
