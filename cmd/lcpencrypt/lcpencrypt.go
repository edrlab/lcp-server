// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpencrypt encrypts publications for use by an LCP Server

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

// LCP Encrypt configuration
type Config struct {
	InputPath    string `split_words:"true"`
	ProviderUri  string `split_words:"true"`
	UUID         string
	GenAltID     bool `split_words:"true"`
	Verbose      bool
	V2           bool
	ExtractCover bool
	PDFNoMeta    bool   `split_words:"true"`
	StoragePath  string `split_words:"true"`
	StorageUrl   string `split_words:"true"`
	LCPServerUrl string `envconfig:"lcpserver_url"`
	CMSUrl       string `split_words:"true" envconfig:"cms_url"`
}

// create an enum with two values: keep_file and delete_file
type FileHandling int

const (
	KeepFile FileHandling = iota
	DeleteFile
)

func init() {
	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

func usage() {
	fmt.Println("Usage: lcpencrypt [-serve] [-input] [-uuid] [-usefn] [-verbose] [-v2] [-pdfnometa] [-cover]")
	flag.PrintDefaults()
}

func main() {

	// parse the command line
	// some values (storage path, storage url, lcp server and cms url) can only be set through environment variables
	serve := flag.Bool("serve", false, "if set, start the utility as a server; the uuid flag is ignored in this mode")
	input := flag.String("input", "", "source file locator (file path or url)")
	uuid := flag.String("uuid", "", "Forced publication UUID")
	provider := flag.String("provider", "", "provider URI of the publication(s)")
	verbose := flag.Bool("verbose", false, "if set, display info messages; if not set, display only warnings and errors.")
	v2 := flag.Bool("v2", true, "indicates a v2 License server")
	cover := flag.Bool("cover", true, "indicates if a cover should be exported")
	pdfnometa := flag.Bool("pdfnometa", false, "if set, indicates that PDF metadata are omitted")
	genaltid := flag.Bool("altid", false, "if set, the file name is used as an alternative publication ID")
	help := flag.Bool("help", false, "shows information")

	flag.Parse()

	if *help {
		usage()
		os.Exit(1)
	}

	var c Config

	// init config from command line flags
	// TODO: Move provider URI and input path to a map in config.
	c.ProviderUri = *provider
	c.InputPath = filepath.Dir(*input)
	c.UUID = *uuid
	c.GenAltID = *genaltid
	c.Verbose = *verbose
	c.V2 = *v2
	c.ExtractCover = *cover
	c.PDFNoMeta = *pdfnometa

	// TODO: Move provider URI and input path to a map as LCPENCRYPT_PROVIDERS="prov1:path1, prov2:path2"
	// UUID makes no sense as an environment variable.
	// INPUT_PATH must be a directory when set as an environment variable.
	// The following environment variables are supported:
	// LCPENCRYPT_INPUT_PATH
	// LCPENCRYPT_PROVIDER_URI
	// LCPENCRYPT_VERBOSE
	// LCPENCRYPT_V2
	// LCPENCRYPT_COVER
	// LCPENCRYPT_PDF_NO_META
	// LCPENCRYPT_GEN_ALT_ID
	// LCPENCRYPT_STORAGE_PATH
	// LCPENCRYPT_STORAGE_URL
	// LCPENCRYPT_LCPSERVER_URL
	// LCPENCRYPT_CMS_URL
	// process environment variables
	err := envconfig.Process("lcpencrypt", &c)
	if err != nil {
		log.Errorln("Configuration failed: " + err.Error())
		os.Exit(1)
	}

	// the verbose flag acts on the info level
	if !c.Verbose {
		log.SetLevel(log.WarnLevel)
	}

	// get the file name from the input path
	filename := filepath.Base(*input)

	if *serve {
		log.Infoln("Entering server mode")
		log.Infoln("Watching directory: ", os.Getenv("LCPENCRYPT_INPUT_PATH"))
		log.Infoln("Storage path: ", os.Getenv("LCPENCRYPT_STORAGE_PATH"))
		// start the utility as a server
		activateServer(c)
	} else if filename != "." {
		// run the utility as a command line tool, keeping the input file in place
		err = processFile(c, filename, KeepFile)
		if err != nil {
			log.Errorf("Error processing file: %v", err)
		}
	} else {
		usage()
		os.Exit(1)
	}

}
