// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package sign

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"math"
	"math/big"
)

type Signature struct {
	Certificate []byte `json:"certificate"`
	Value       []byte `json:"value"`
	Algorithm   string `json:"algorithm"`
}

var SignatureAlgorithm_RSA = "http://www.w3.org/2001/04/xmldsig-more#rsa-sha256"
var SignatureAlgorithm_ECDSA = "http://www.w3.org/2001/04/xmldsig-more#ecdsa-sha256"

// ------
// Signer
// ------

type Signer interface {
	Sign(interface{}) (Signature, error)
}

// Creates a new signer, which depends on the certificate type. Currently supports
// RSA (PKCS1v15) and ECDSA (SHA256 is used in both cases)
func NewSigner(cert *tls.Certificate) (Signer, error) {
	/*
		func NewSigner(certFile, keyFile string) (Signer, error) {
				certData, err := os.ReadFile(certFile)
				if err != nil {
					return nil, errors.New("failed to read the certificate")
				}

				keyData, err := os.ReadFile(keyFile)
				if err != nil {
					return nil, errors.New("failed to read the private key")
				}
	*/

	switch privKey := cert.PrivateKey.(type) {
	case *ecdsa.PrivateKey:
		return &ecdsaSigner{privKey, cert}, nil
	case *rsa.PrivateKey:
		return &rsaSigner{privKey, cert}, nil
	}

	return nil, errors.New("unsupported certificate type")
}

// ECDSA
type ecdsaSigner struct {
	key  *ecdsa.PrivateKey
	cert *tls.Certificate
}

// copyWithLeftPad fills the resulting output according to the XMLDSIG spec
func copyWithLeftPad(dest, src []byte) {
	numPaddingBytes := len(dest) - len(src)
	for i := 0; i < numPaddingBytes; i++ {
		dest[i] = 0
	}
	copy(dest[numPaddingBytes:], src)
}

// Sign signs any json structure
func (signer *ecdsaSigner) Sign(in interface{}) (sig Signature, err error) {
	canon, err := Canon(in)
	if err != nil {
		return
	}

	hash := sha256.Sum256(canon)
	r, s, err := ecdsa.Sign(rand.Reader, signer.key, hash[:])
	if err != nil {
		return
	}

	curveSizeInBytes := int(math.Ceil(float64(signer.key.Curve.Params().BitSize) / 8))

	// The resulting signature is the concatenation of the big-endian octet strings
	// of the r and s parameters, each padded to the byte size of the curve order.
	sig.Value = make([]byte, 2*curveSizeInBytes)
	copyWithLeftPad(sig.Value[0:curveSizeInBytes], r.Bytes())
	copyWithLeftPad(sig.Value[curveSizeInBytes:], s.Bytes())

	sig.Algorithm = SignatureAlgorithm_ECDSA
	sig.Certificate = signer.cert.Certificate[0]
	return
}

// RSA
type rsaSigner struct {
	key  *rsa.PrivateKey
	cert *tls.Certificate
}

// Sign returns a signature for the provided json
func (signer *rsaSigner) Sign(in interface{}) (sig Signature, err error) {
	canon, err := Canon(in)
	if err != nil {
		return
	}

	hash := sha256.Sum256(canon)
	sig.Value, err = rsa.SignPKCS1v15(rand.Reader, signer.key, crypto.SHA256, hash[:])
	if err != nil {
		return
	}

	sig.Algorithm = SignatureAlgorithm_RSA
	sig.Certificate = signer.cert.Certificate[0]
	return
}

// -----------
// SignChecker
// -----------
// Because SignChecker is generic,
// the embedded signature of an LCP license must have been removed before the call

// SignChecker is the interface allowing the verification of a signature
type SignChecker interface {
	Check(interface{}, []byte) error
}

// NewSignChecker creates a new signature checker, which depends on the certificate type (RSA or ECDSA)
func NewSignChecker(certData []byte, certType string) (SignChecker, error) {

	//fmt.Println("Certificate")
	//fmt.Println(b64.StdEncoding.EncodeToString(certData))

	// parse the provider certificate (as ASN.1 DER data)
	cert, err := x509.ParseCertificate(certData)
	if err != nil {
		return nil, errors.New("failed to parse the certificate")
	}

	// generate a typed signature checker
	switch pubKey := cert.PublicKey.(type) {
	case *ecdsa.PublicKey:
		if certType != SignatureAlgorithm_ECDSA {
			return nil, errors.New("invalid signature algorithm; ECDSA was expected")
		}
		return &ecdsaSignChecker{pubKey}, nil
	case *rsa.PublicKey:
		if certType != SignatureAlgorithm_RSA {
			return nil, errors.New("invalid signature algorithm; RSA was expected")
		}
		return &rsaSignChecker{pubKey}, nil
	}

	return nil, errors.New("unsupported certificate type")
}

// ECDSA
type ecdsaSignChecker struct {
	key *ecdsa.PublicKey
}

// Check verifies the signature of any json structure
func (checker *ecdsaSignChecker) Check(in interface{}, signature []byte) (err error) {
	// make the structure canonical
	plain, err := Canon(in)
	if err != nil {
		return
	}

	// generate a hash
	hash := sha256.Sum256(plain)

	// retrieve the signature vectors
	r := new(big.Int).SetBytes(signature[:len(signature)/2])
	s := new(big.Int).SetBytes(signature[len(signature)/2:])

	// check the hash vs the public key and signature
	if !ecdsa.Verify(checker.key, hash[:], r, s) {
		return errors.New("failed to verify the signature")
	}
	return nil
}

// RSA
type rsaSignChecker struct {
	key *rsa.PublicKey
}

// Check verifies the signature of any json structure
func (checker *rsaSignChecker) Check(in interface{}, signature []byte) (err error) {
	// make the structure canonical
	canon, err := Canon(in)
	if err != nil {
		return
	}

	hash := sha256.Sum256(canon)

	// check the hash vs the public key and signature
	err = rsa.VerifyPKCS1v15(checker.key, crypto.SHA256, hash[:], signature)
	if err != nil {
		return
	}
	return nil
}
