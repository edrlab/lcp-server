// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package conf

import (
	"log"
	"os"
	"path/filepath"

	"github.com/kelseyhightower/envconfig"
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
	Username string `yaml:"username" envconfig:"access_username"`
	Password string `yaml:"password" envconfig:"access_password"`
}

type Certificate struct {
	Cert       string `yaml:"cert" envconfig:"certificate_cert"`              // Path
	PrivateKey string `yaml:"private_key" envconfig:"certificate_privatekey"` // Path
}

type License struct {
	Provider string `yaml:"provider"  envconfig:"license_provider"`   // URI
	Profile  string `yaml:"profile"  envconfig:"license_profile"`     // standard profile URI
	HintLink string `yaml:"hint_link"  envconfig:"license_hint_link"` // URL
}

type Status struct {
	FreshLicenseLink            string `yaml:"fresh_license_link" envconfig:"status_freshlicenselink"`
	AllowRenewOnExpiredLicenses bool   `yaml:"allow_renew_on_expired_licenses" envconfig:"status_allowrenewonexpiredlicenses"`
	RenewDefaultDays            int    `yaml:"renew_default_days" envconfig:"status_renewdefaultdays"`
	RenewMaxDays                int    `yaml:"renew_max_days" envconfig:"status_renewmaxdays"`
	RenewLink                   string `yaml:"renew_link" envconfig:"status_renewlink"`
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
		log.Println("failed to find the configuration file")
	}

	err := envconfig.Process("lcpserver", &c)
	if err != nil {
		return &c, err
	}

	// Set some defaults
	if c.Port == 0 {
		c.Port = 8081
	}

	return &c, nil
}
