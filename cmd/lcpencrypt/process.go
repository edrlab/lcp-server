// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpencrypt server mode

package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/readium/readium-lcp-server/encrypt"
)

// processFile processes a single file.
// rawInput is the original -input value (full URL or local path); filename is filepath.Base of that.
func processFile(c Config, rawInput string, filename string, fileHandling FileHandling) error {
	log.Printf("Processing file: %s", rawInput)

	inputFilePath, cleanup, err := processRemoteFile(rawInput, filename, c.InputPath)
	if err != nil {
		return err
	}
	defer cleanup()

	// extract the username and password from the url, remove them from the url
	var username, password string
	err = getUsernamePassword(&c.LCPServerUrl, &username, &password)
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

	log.Println("Starting encryption...")
	workDir, cleanupWorkDir, err := createWorkDir()
	if err != nil {
		return err
	}
	defer cleanupWorkDir()
	// @TODO: Remove this hardcoding and make it configurable
	storageFilename := "encrypted/" + c.UUID

	publication, err := encrypt.ProcessEncryption(c.UUID, contentkey, inputFilePath, workDir, "", c.StoragePath, c.StorageUrl, storageFilename, c.ExtractCover, c.PDFNoMeta)
	if err != nil {
		return err
	}

	// Set the publication AltID (if extracted from the filename or imposed)
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

	if fileHandling == DeleteFile && !isRemoteURL(rawInput) {
		if err := os.Remove(inputFilePath); err != nil {
			return err
		}
		log.Printf("Input file deleted: %s", filename)
	}
	return nil
}

// isRemoteURL checks if a string is a remote URL
func isRemoteURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") || strings.HasPrefix(s, "ftp://")
}

// createWorkDir creates a temporary working directory with an "encrypted"
// subdirectory so the encrypted output file (<uuid>.epub) never collides
// with the input file. The returned cleanup function removes the entire tree.
func createWorkDir() (workDir string, cleanup func(), err error) {
	workDir, err = os.MkdirTemp("", "lcpencrypt-*")
	if err != nil {
		return "", nil, fmt.Errorf("creating work directory: %w", err)
	}

	if err := os.MkdirAll(filepath.Join(workDir, "encrypted"), os.ModePerm); err != nil {
		os.RemoveAll(workDir)
		return "", nil, fmt.Errorf("creating encrypted dir: %w", err)
	}

	return workDir, func() { os.RemoveAll(workDir) }, nil
}

// processRemoteFile resolves the input to a local file path. For remote URLs,
// it downloads the file to a randomly-named temp file so that the library's
// internal temp file (named <contentID>.epub) never collides with it.
// The returned cleanup function removes the temp file; for local inputs it is a no-op.
func processRemoteFile(rawInput, filename, inputPath string) (localPath string, cleanup func(), err error) {
	if !isRemoteURL(rawInput) {
		return filepath.Join(inputPath, filename), func() {}, nil
	}

	tmpFile, err := os.CreateTemp("", "lcpinput-*"+filepath.Ext(filename))
	if err != nil {
		return "", nil, fmt.Errorf("creating temp file for download: %w", err)
	}

	resp, err := http.Get(rawInput)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("downloading %s: %w", rawInput, err)
	}
	_, copyErr := io.Copy(tmpFile, resp.Body)
	resp.Body.Close()
	tmpFile.Close()
	if copyErr != nil {
		os.Remove(tmpFile.Name())
		return "", nil, fmt.Errorf("saving downloaded file: %w", copyErr)
	}

	log.Debugf("Downloaded %s → %s", rawInput, tmpFile.Name())
	return tmpFile.Name(), func() { os.Remove(tmpFile.Name()) }, nil
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
