// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package conf

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

// LCP Server configuration
type Config struct {
	PublicBaseUrl string `yaml:"public_base_url"`
	Port          int    `yaml:"port"`
	Dsn           string `yaml:"dsn"`
	Login         `yaml:"login"`
	Certificate   `yaml:"certificate"`
	License       `yaml:"license"`
	Status        `yaml:"status"`
}

type Login struct {
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Certificate struct {
	Cert       string `yaml:"cert"`
	PrivateKey string `yaml:"private_key"`
}

type License struct {
	Provider string `yaml:"provider"` // URI
	Profile  string `yaml:"profile"`  // "http://readium.org/lcp/basic-profile" || "http://readium.org/lcp/profile-1.0" || ...
	HintLink string `yaml:"hint_links"`
}

type Status struct {
	RenewDefaultDays int    `yaml:"renew_default_days"`
	RenewMaxDays     int    `yaml:"renew_max_days"`
	RenewLink        string `yaml:"renew_link"`
}

func ReadConfig(configFile string) (*Config, error) {

	var c Config

	if configFile != "" {
		f, _ := filepath.Abs(configFile)
		yamlData, err := os.ReadFile(f)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(yamlData, &c)
		if err != nil {
			return nil, err
		}

	} else {
		return nil, errors.New("failed to find the configuration file")
	}

	return &c, nil
}
