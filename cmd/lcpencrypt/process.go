// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpencrypt server mode

package main

import (
	"fmt"
	"net/url"
	"os"
	"path"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/readium/readium-lcp-server/encrypt"
)

// processFile processes a single file
func processFile(c Config, fileName string) error {
	log.Printf("Processing file: %s", fileName)

	// create a path from c.InputPath and fileName
	inputFilePath := path.Join(c.InputPath, fileName)

	// extract the username and password from the url, remove them from the url
	var username, password string
	err := getUsernamePassword(&c.LCPServerUrl, &username, &password)
	if err != nil {
		return err
	}

	// if contentid is set, check if the content already exists in the License Server.
	// If this is the case, get the content encryption key for the server, so that the new encryption
	// keeps the same key.
	// This is necessary to allow fresh licenses being capable of decrypting previously downloaded content.
	var contentkey string
	if c.UUID != "" {
		// warning: this is a synchronous REST call
		// contentKey is not initialized if the content does not exist in the License Server
		contentkey, err = getContentKey(c.UUID, c.LCPServerUrl, username, password, c.V2)
		if err != nil {
			return err
		}
	}

	// if the file name is used as storage file name, use it without extension
	var storageFileName string
	if c.UseFileName {
		storageFileName = fileName
		log.Println("Storage file name:", storageFileName)
	}

	start := time.Now()

	// encrypt the publication
	// no specific temp directory, no specific output directory
	// request a cover image
	log.Println("Starting encryption...")
	publication, err := encrypt.ProcessEncryption(c.UUID, contentkey, inputFilePath, "", "", c.StoragePath, c.StorageUrl, storageFileName, true)
	if err != nil {
		return err
	}

	if c.LCPServerUrl == "" {
		// If no LCP server URL is provided, we can't notify the server
		log.Println("No LCP server URL provided, skipping notification.")
		return nil
	}

	elapsed := time.Since(start)

	// notify the license server
	err = encrypt.NotifyLCPServer(*publication, c.ProviderUri, c.LCPServerUrl, c.V2, username, password, c.Verbose)
	if err != nil {
		return err
	}

	// notify a CMS (username and password are always in the URL)
	err = encrypt.NotifyCMS(*publication, c.CMSUrl, c.Verbose)
	if err != nil {
		fmt.Println("Error notifying the CMS:", err.Error())
		// abort the notification of the license server
		err = encrypt.AbortNotification(*publication, c.LCPServerUrl, c.V2, username, password)
		if err != nil {
			return err
		}
	}

	fmt.Println("The encryption took", elapsed)

	// delete the file
	if err := os.Remove(inputFilePath); err != nil {
		return err
	}
	log.Printf("File deleted: %s", fileName)
	return nil
}

// getUsernamePassword looks for the username and password in the url
func getUsernamePassword(notifyURL, username, password *string) error {
	u, err := url.Parse(*notifyURL)
	if err != nil {
		return err
	}
	un := u.User.Username()
	pw, pwfound := u.User.Password()
	if un != "" && pwfound {
		*username = un
		*password = pw
		u.User = nil
		*notifyURL = u.String() // notifyURL is updated
	}
	return nil
}
