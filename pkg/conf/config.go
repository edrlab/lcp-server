// Copyright 2024 European Digital Reading Lab. All rights reserved.
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
	LogLevel      string `yaml:"log_level"` // "debug", "info", "warn", "error"
	PublicBaseUrl string `yaml:"public_base_url"`
	Port          int    `yaml:"port"`
	Dsn           string `yaml:"dsn"`
	Access        `yaml:"access"`
	Certificate   `yaml:"certificate"`
	License       `yaml:"license"`
	Status        `yaml:"status"`
	Resources     string `yaml:"resources"`
}

type Access struct {
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type Certificate struct {
	Cert       string `yaml:"cert"`
	PrivateKey string `yaml:"private_key"`
}

type License struct {
	Provider string `yaml:"provider"` // URI
	Profile  string `yaml:"profile"`  // "http://readium.org/lcp/basic-profile" || "http://readium.org/lcp/profile-1.0" || ...
	HintLink string `yaml:"hint_link"`
}

type Status struct {
	FreshLicenseLink            string `yaml:"fresh_license_link"`
	AllowRenewOnExpiredLicenses bool   `yaml:"allow_renew_on_expired_licenses"`
	RenewDefaultDays            int    `yaml:"renew_default_days"`
	RenewMaxDays                int    `yaml:"renew_max_days"`
	RenewLink                   string `yaml:"renew_link"`
}

func Init(configFile string) (*Config, error) {

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

	if c.Port == 0 {
		c.Port = 8081
	}

	return &c, nil
}
