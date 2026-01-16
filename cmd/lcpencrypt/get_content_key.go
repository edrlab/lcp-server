// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/edrlab/lcp-server/pkg/stor"

	log "github.com/sirupsen/logrus"
)

type ContentInfo struct {
	ID            string `json:"id"`
	EncryptionKey []byte `json:"key,omitempty"`
}

type PubInfo struct {
	UUID            string `json:"uuid"`
	EncryptionKey []byte   `json:"encryption_key"`
}

// getContentKey gets content information from the License Server
// for a given content id, and returns the associated content key.
func getContentKey(pubID, AltID, lcpsv, username, password string, v2 bool) (string, string, error) {

	var contentKey string

	// An empty notify URL is not an error, simply a test encryption
	if lcpsv == "" {
		return contentKey, pubID, nil
	}
	if pubID == "" && AltID == "" {
		return contentKey, pubID, errors.New("publication ID or AltID must be set to get content information from the LCP Server")
	}

	log.Debug("Checking if the publication exists on the LCP Server...")

	if !strings.HasPrefix(lcpsv, "http://") && !strings.HasPrefix(lcpsv, "https://") {
		lcpsv = "http://" + lcpsv
	}
	var getInfoURL string
	var err error
	if v2 {
		if pubID != "" {
			getInfoURL, err = url.JoinPath(lcpsv, "publications", pubID)
		} else if AltID != "" {
			getInfoURL, err = url.JoinPath(lcpsv, "publications/altid", AltID)
		} 
	} else {
			getInfoURL, err = url.JoinPath(lcpsv, "contents", pubID, "info")
	}
	if err != nil {
		return contentKey, pubID, err
	}

	req, err := http.NewRequest("GET", getInfoURL, nil)
	if err != nil {
		return contentKey, pubID, err
	}

	req.SetBasicAuth(username, password)
	client := &http.Client{
		Timeout: 15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return contentKey, pubID, err
	}
	defer resp.Body.Close()

	// if the content is found, the content key is updated
	if resp.StatusCode == http.StatusOK {
		if v2 {
			log.Debug("Publication found on LCP Server (v2)")
			publication := stor.Publication{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&publication)
			if err != nil {
				return contentKey, pubID, errors.New("unable to decode content information")
			}
			contentKey = b64.StdEncoding.EncodeToString(publication.EncryptionKey)
			pubID = publication.UUID
		} else {
			log.Debug("Publication found on LCP Server (v1)")
			contentInfo := ContentInfo{}
			dec := json.NewDecoder(resp.Body)
			err = dec.Decode(&contentInfo)
			if err != nil {
				return contentKey, pubID, errors.New("unable to decode content information")
			}
			contentKey = b64.StdEncoding.EncodeToString(contentInfo.EncryptionKey)
			pubID = contentInfo.ID
		}
		log.Debug("Existing encryption key retrieved, uuid ", pubID)
	} else {
		log.Debug("Content not found on LCP Server, a new encryption key will be generated")
	}
	return contentKey, pubID, nil
}
