// Copyright 2023 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// lcpchecker validates LCP licenses against the specification

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/edrlab/lcp-server/pkg/check"
	log "github.com/sirupsen/logrus"
)

func init() {
	// Output to stdout instead of the default stderr
	log.SetOutput(os.Stdout)

	log.SetFormatter(&log.TextFormatter{
		DisableTimestamp: true,
	})
}

func usage() {
	fmt.Println("Usage: lcpchecker [-passphrase] [-level] [-verbose] filepath")
	flag.PrintDefaults()
}

func main() {

	// parse the command line
	passphrase := flag.String("passphrase", "", "user passphrase")
	level := flag.Uint("level", 0, "checker level (1 = default, up to 3)")
	verbose := flag.Bool("verbose", false, "display positive tests")
	flag.Parse()

	values := flag.Args()
	if len(values) == 0 {
		usage()
		os.Exit(1)
	}
	filepath := values[0]

	// log the file name
	fmt.Println("Checking ", filepath)

	// the verbose flag acts on the info level
	if !*verbose {
		log.SetLevel(log.WarnLevel)
	}

	// open the license
	bytes, err := os.ReadFile(filepath)
	if err != nil {
		log.Error("Error: ", err)
		os.Exit(1)
	}

	// pass all checks
	err = check.Checker(bytes, *passphrase, *level)
	if err != nil {
		log.Error(err)
	}
}
