// Copyright 2025 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

package conf

import (
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/kelseyhightower/envconfig"
	"gopkg.in/yaml.v2"
)

// LCP Server configuration
type Config struct {
	LogLevel      string `yaml:"log_level" envconfig:"loglevel"` // "debug", "info", "warn", "error"
	PublicBaseUrl string `yaml:"public_base_url" envconfig:"publicbaseurl"`
	Port          int    `yaml:"port"`
	Dsn           string `yaml:"dsn"`
	Access        `yaml:"access"`
	Certificate   `yaml:"certificate"`
	License       `yaml:"license"`
	Status        `yaml:"status"`
	Dashboard     `yaml:"dashboard"`
	JWT           `yaml:"jwt"`
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
	Provider string `yaml:"provider"  envconfig:"license_provider"`  // URI
	Profile  string `yaml:"profile"  envconfig:"license_profile"`    // default profile URI
	HintLink string `yaml:"hint_link"  envconfig:"license_hintlink"` // URL
}

type Status struct {
	FreshLicenseLink            string `yaml:"fresh_license_link" envconfig:"status_freshlicenselink"`
	AllowRenewOnExpiredLicenses bool   `yaml:"allow_renew_on_expired_licenses" envconfig:"status_allowrenewonexpiredlicenses"`
	RenewDefaultDays            int    `yaml:"renew_default_days" envconfig:"status_renewdefaultdays"`
	RenewMaxDays                int    `yaml:"renew_max_days" envconfig:"status_renewmaxdays"`
	RenewLink                   string `yaml:"renew_link" envconfig:"status_renewlink"`
}

type Dashboard struct {
	ExcessiveSharingThreshold int  `yaml:"excessive_sharing_threshold" envconfig:"dashboard_excessivesharingthreshold"`
	LimitToLast12Months       bool `yaml:"limit_to_last_12_months" envconfig:"dashboard_limittolast12months"`
}

type JWT struct {
	SecretKey string            `yaml:"secret_key" envconfig:"jwt_secretkey"`
	Admin     map[string]string `yaml:"admin" envconfig:"jwt_admin"` // list of admin usernames and passwords
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
		log.Println("Configuration file", configFile)
	} else {
		log.Println("No configuration file provided, using environment variables and defaults")
	}

	// Process environment variables
	err := envconfig.Process("lcpserver", &c)
	if err != nil {
		return &c, err
	}

	// Process docker secrets
	if os.Getenv("LCPSERVER_ACCESS_FILE") != "" {
		data, err := os.ReadFile(os.Getenv("LCPSERVER_ACCESS_FILE"))
		if err != nil {
			return nil, err
		}
		lines := strings.Split(string(data), "\n")
		var tuple []string
		if len(lines) >= 1 {
			// the first line is the username and password used to access the server using basic auth (private routes).
			// it overrides any value set in the configuration file or environment variables.
			tuple = strings.Split(string(lines[0]), ":")
			if len(tuple) == 2 {
				c.Access.Username = strings.TrimSpace(tuple[0])
				c.Access.Password = strings.TrimSpace(tuple[1])
			}
		}
		if len(lines) >= 2 {
			// next lines are usernames and passwords used to access the server using JWT (dashboard).
			// it completes any value set in the configuration file or environment variables.
			for _, line := range lines[1:] {
				tuple = strings.Split(string(line), ":")
				if len(tuple) == 2 {
					username := strings.TrimSpace(tuple[0])
					password := strings.TrimSpace(tuple[1])
					if username != "" && password != "" {
						// Initialize map if nil
						if c.JWT.Admin == nil {
							c.JWT.Admin = make(map[string]string)
						}
						c.JWT.Admin[username] = password
					}
				}
			}
		}
	}

	// Process MySQL password from Docker secret
	if mysqlPasswordFile := os.Getenv("MYSQL_PASSWORD_FILE"); mysqlPasswordFile != "" {
		data, err := os.ReadFile(mysqlPasswordFile)
		if err != nil {
			log.Printf("Could not read MySQL password from %s: %v", mysqlPasswordFile, err)
		} else {
			mysqlPassword := strings.TrimSpace(string(data))
			if mysqlPassword != "" && c.Dsn != "" {
				// Replace password placeholder in DSN
				// Supports format: mysql://user:PASSWORD_PLACEHOLDER@host/db or mysql://user:oldpassword@host/db
				c.Dsn = strings.Replace(c.Dsn, "PASSWORD_PLACEHOLDER", mysqlPassword, 1)
				// Also try to replace any existing password in the DSN (between : and @)
				if strings.Contains(c.Dsn, "mysql://") {
					parts := strings.SplitN(c.Dsn, ":", 3)
					if len(parts) == 3 {
						// parts[0] = "mysql"
						// parts[1] = "//user"
						// parts[2] = "oldpassword@host/db"
						atIndex := strings.Index(parts[2], "@")
						if atIndex >= 0 {
							c.Dsn = parts[0] + ":" + parts[1] + ":" + mysqlPassword + parts[2][atIndex:]
						}
					}
				}
				log.Println("MySQL password loaded from Docker secret")
			}
		}
	}

	// Process JWT secret key from Docker secret
	if jwtSecretFile := os.Getenv("JWT_SECRETKEY_FILE"); jwtSecretFile != "" {
		data, err := os.ReadFile(jwtSecretFile)
		if err != nil {
			log.Printf("Could not read JWT secret key from %s: %v", jwtSecretFile, err)
		} else {
			jwtSecret := strings.TrimSpace(string(data))
			if jwtSecret != "" {
				c.JWT.SecretKey = jwtSecret
				log.Println("JWT secret key loaded from Docker secret")
			}
		}
	}

	// Set some defaults
	if c.Port == 0 {
		c.Port = 8989
	}
	if c.Dashboard.ExcessiveSharingThreshold == 0 {
		c.Dashboard.ExcessiveSharingThreshold = 1
	}
	if c.JWT.SecretKey == "" {
		c.JWT.SecretKey = "default_jwt_secret_key_please_change_in_production"
	}

	// Initialize JWT.Admin map if nil
	if c.JWT.Admin == nil {
		c.JWT.Admin = make(map[string]string)
	}

	// Set default dashboard account if none configured
	if len(c.JWT.Admin) == 0 {
		c.JWT.Admin["admin"] = "supersecret"
		log.Println("⚠️  No dashboard account configured, using default account: admin/supersecret")
	}

	// Log configured dashboard accounts (without passwords for security)
	log.Printf("📋 Configured dashboard accounts: %d", len(c.JWT.Admin))
	for name := range c.JWT.Admin {
		log.Printf("   - %s", name)
	}

	return &c, nil
}
