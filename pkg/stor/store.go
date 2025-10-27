// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// The stor package manages the storage of our entities.
package stor

import (
	"fmt"
	"os"
	"strings"
	"time"

	"log"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type (

	// generic store
	dbStore struct {
		db *gorm.DB
	}

	// entity stores
	publicationStore dbStore
	licenseStore     dbStore
	eventStore       dbStore
	dashboardStore   dbStore

	// Store interface, giving access to specialized interfaces
	Store interface {
		Publication() PublicationRepository
		License() LicenseRepository
		Event() EventRepository
		Dashboard() DashboardRepository
	}

	// PublicationRepository interface, defining publication operations
	PublicationRepository interface {
		ListAll() (*[]Publication, error)
		List(pageNum, pageSize int) (*[]Publication, error)
		FindByType(contentType string) (*[]Publication, error)
		Count() (int64, error)
		Get(uuid string) (*Publication, error)
		Create(p *Publication) error
		Update(p *Publication) error
		Delete(p *Publication) error
	}

	// LicenseRepository interface, defining license operations
	LicenseRepository interface {
		ListAll() (*[]LicenseInfo, error)
		List(pageNum, pageSize int) (*[]LicenseInfo, error)
		FindByUser(userID string) (*[]LicenseInfo, error)
		FindByPublication(publicationID string) (*[]LicenseInfo, error)
		FindByStatus(status string) (*[]LicenseInfo, error)
		FindByDeviceCount(min int, max int) (*[]LicenseInfo, error)
		Count() (int64, error)
		Get(uuid string) (*LicenseInfo, error)
		Create(p *LicenseInfo) error
		Update(p *LicenseInfo) error
		Delete(p *LicenseInfo) error
	}

	// EventRepository interface, defining event operations
	EventRepository interface {
		List(licenseID string) (*[]Event, error)
		GetRegisterByDevice(licenseID string, deviceID string) (*Event, error)
		Count(licenseID string) (int64, error)
		Get(id uint) (*Event, error)
		Create(e *Event) error
		Update(e *Event) error
		Delete(e *Event) error
	}

	// DashboardRepository interface, defining dashboard operations
	DashboardRepository interface {
		GetDashboard(excessiveSharingThreshold int, limitToLast12Months bool) (*DashboardData, error)
		GetOversharedLicenses(excessiveSharingThreshold int, limitToLast12Months bool) ([]OversharedLicenseData, error)
	}
)

// implementation of the different repository interfaces
func (s *dbStore) Publication() PublicationRepository {
	return (*publicationStore)(s)
}

func (s *dbStore) License() LicenseRepository {
	return (*licenseStore)(s)
}

func (s *dbStore) Event() EventRepository {
	return (*eventStore)(s)
}

// Dashboard implements Store.
func (s *dbStore) Dashboard() DashboardRepository {
	return (*dashboardStore)(s)
}

// List of status values as strings
const (
	STATUS_READY     = "ready"
	STATUS_ACTIVE    = "active"
	STATUS_REVOKED   = "revoked"
	STATUS_RETURNED  = "returned"
	STATUS_CANCELLED = "cancelled"
	STATUS_EXPIRED   = "expired"
	EVENT_REGISTER   = "register"
	EVENT_RENEW      = "renew"
	EVENT_RETURN     = "return"
	EVENT_REVOKE     = "revoke"
	EVENT_CANCEL     = "cancel"
)

// Init initializes the database
func Init(dsn string) (Store, error) {
	var err error

	dialect, cnx := dbFromURI(dsn)
	if dialect == "error" {
		return nil, fmt.Errorf("incorrect database source name: %q", dsn)
	}

	// add parameters specific to the dialect
	cnx = addParamsDialectSpecific(cnx, dialect)

	// database logger
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             time.Second, // Slow SQL threshold
			LogLevel:                  logger.Warn, // Log level (Silent, Error, Warn, Info)
			IgnoreRecordNotFoundError: true,        // Ignore ErrRecordNotFound error for logger
			Colorful:                  true,        // Enable color
		},
	)

	db, err := gorm.Open(GormDialector(cnx), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		log.Printf("Failed connecting to the database: %v", err)
		return nil, err
	}

	err = performDialectSpecific(db, dialect)
	if err != nil {
		log.Printf("Failed performing dialect specific database init: %v", err)
		return nil, err
	}

	err = db.AutoMigrate(&Publication{}, &LicenseInfo{}, &Event{})
	if err != nil {
		log.Printf("Failed performing database automigrate: %v", err)
		return nil, err
	}

	stor := &dbStore{db: db}

	return stor, nil
}

// dbFromURI
func dbFromURI(uri string) (string, string) {
	parts := strings.Split(uri, "://")
	if len(parts) != 2 {
		return "error", ""
	}
	return parts[0], parts[1]
}

// addParamsDialectSpecific takes a connection string and adds parameters specific to the SQL dialect
func addParamsDialectSpecific(cnx, dialect string) string {
	switch dialect {
	case "sqlite3":
		cnx += "?cache=shared&mode=rwc"
	case "mysql":
		cnx += "?charset=utf8mb4&parseTime=True&loc=Local"
	case "postgres":
		cnx += "?sslmode=disable"
	case "mssql":
		// nothing , so far
	default:
		log.Printf("Invalid dialect: %s", dialect)
	}
	return cnx
}

// performDialectSpecific
func performDialectSpecific(db *gorm.DB, dialect string) error {
	switch dialect {
	case "sqlite3":
		err := db.Exec("PRAGMA journal_mode = WAL").Error
		if err != nil {
			return err
		}
		err = db.Exec("PRAGMA foreign_keys = ON").Error
		if err != nil {
			return err
		}
	case "mysql":
		// nothing , so far
	case "postgres":
		// nothing , so far
	case "mssql":
		// nothing , so far
	default:
		return fmt.Errorf("invalid dialect: %s", dialect)
	}
	return nil
}
