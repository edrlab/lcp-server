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
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/readium/readium-lcp-server/encrypt"
)

// processFile processes a single file
func processFile(c Config, filename string, fileHandling FileHandling) error {
	log.Printf("Processing file: %s", filename)

	// create a path from c.InputPath and filename
	inputFilePath := path.Join(c.InputPath, filename)

	// extract the username and password from the url, remove them from the url
	var username, password string
	err := getUsernamePassword(&c.LCPServerUrl, &username, &password)
	if err != nil {
		return err
	}

	// if the publication UUID or AltID is imposed, check if the content already exists in the License Server.
	// Note that the publication UUID or AltID may also have be set via the command line. 
	// If this is the case, get the content encryption key for the server, so that the new encryption
	// keeps the same key.
	// This is necessary to allow fresh licenses being capable of decrypting previously downloaded content.
	filen := strings.TrimSuffix(filename, filepath.Ext(filename))
	switch c.UseFilenameAs {
	case "uuid":
		c.UUID = filen
		c.AltID = ""
	case "altid":
		c.AltID = filen
		c.UUID = ""
	}

	var contentkey, uuid string
	if c.UUID != "" || c.AltID != "" {
		// warning: this is a synchronous REST call
		// contentKey and uuid are not initialized if the content does not exist in the License Server
		contentkey, uuid, err = getContentKey(c.UUID, c.AltID, c.LCPServerUrl, username, password, c.V2)
		if err != nil {
			return err
		}
		// set the publication UUID if returned by the server
		if uuid != "" {
			c.UUID = uuid
		}
	}

	start := time.Now()

	// encrypt the publication
	// no specific temp directory, no specific output directory
	// request a cover image
	log.Println("Starting encryption...")
	publication, err := encrypt.ProcessEncryption(c.UUID, contentkey, inputFilePath, "", "", c.StoragePath, c.StorageUrl, "", c.ExtractCover, c.PDFNoMeta)
	if err != nil {
		return err
	}

	// temporary: override publication.AltID (set to the filename by process encryption) - to be suppress with lcpencrypt 1.12.8 
	publication.AltID = c.AltID

	if c.LCPServerUrl == "" {
		// If no LCP server URL is provided, we can't notify the server
		log.Println("No LCP server URL provided, skipping notification.")
		return nil
	}

	elapsed := time.Since(start)

	// notify the license server
	err = encrypt.NotifyLCPServer(*publication, contentkey != "", c.ProviderUri, c.LCPServerUrl, c.V2, username, password, c.Verbose)
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

	if fileHandling == DeleteFile {
		// delete the file
		if err := os.Remove(inputFilePath); err != nil {
			return err
		}
		log.Printf("Input file deleted: %s", filename)
	}
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
