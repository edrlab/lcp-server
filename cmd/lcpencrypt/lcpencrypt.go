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
	ProviderUri  string `split_words:"true"`
	InputPath    string `split_words:"true"`
	UUID         string
	UseFileName  bool   `split_words:"true" envconfig:"usefn"`
	StoragePath  string `split_words:"true"`
	StorageUrl   string `split_words:"true"`
	LCPServerUrl string `envconfig:"lcpserver_url"`
	CMSUrl       string `split_words:"true" envconfig:"cms_url"`
	Verbose      bool
	V2           bool
}

func init() {
	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

func usage() {
	fmt.Println("Usage: lcpencrypt [-serve] [-input] [-uuid] [-usefn] [-verbose] [-v2]")
	flag.PrintDefaults()
}

func main() {

	// parse the command line
	serve := flag.Bool("serve", false, "if set, start the utility as a server")
	input := flag.String("input", "", "source file locator (file path or url); only used in command line")
	uuid := flag.String("uuid", "", "force the publication uuid; only used in command line")
	usefn := flag.Bool("usefn", false, "if set, use the input file name as storage file name")
	verbose := flag.Bool("verbose", false, "if set, display info messages; if not set, display only warnings and errors.")
	v2 := flag.Bool("v2", false, "optional, boolean, indicates a v2 License server, true by default")
	help := flag.Bool("help", false, "shows information")

	flag.Parse()

	if *help {
		usage()
		os.Exit(1)
	}

	var c Config

	// init config from command line flags
	c.InputPath = filepath.Dir(*input)
	filename := filepath.Base(*input) // get the file name from the input path
	c.UUID = *uuid
	c.UseFileName = *usefn
	c.Verbose = *verbose
	c.V2 = *v2

	// process environment variables
	// LCPENCRYPT_PROVIDER_URI
	// LCPENCRYPT_USEFN
	// LCPENCRYPT_INPUT_PATH
	// LCPENCRYPT_STORAGE_PATH
	// LCPENCRYPT_STORAGE_URL
	// LCPENCRYPT_LCPSERVER_URL
	// LCPENCRYPT_CMS_URL
	// LCPENCRYPT_VERBOSE
	// LCPENCRYPT_V2
	err := envconfig.Process("lcpencrypt", &c)
	if err != nil {
		log.Errorln("Configuration failed: " + err.Error())
		os.Exit(1)
	}

	// the verbose flag acts on the info level
	if !c.Verbose {
		log.SetLevel(log.WarnLevel)
	}

	if *serve {
		log.Warnln("Entering server mode")
		log.Infoln("Watching directory: ", os.Getenv("LCPENCRYPT_INPUT_PATH"))
		log.Infoln("Storage path: ", os.Getenv("LCPENCRYPT_STORAGE_PATH"))
		// start the utility as a server
		activateServer(c)
	} else {
		// run the utility as a command line tool
		err = processFile(c, filename)
		if err != nil {
			log.Errorf("Error processing file %s: %v", c.InputPath, err)
		}
	}

}
