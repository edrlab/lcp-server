// Copyright 2022 European Digital Reading Lab. All rights reserved.
// Use of this source code is governed by a BSD-style license
// specified in the Github project LICENSE file.

// Package stor manages entity storage.
package stor

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
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

	// Store interface, giving access to specialized interfaces
	Store interface {
		Publication() PublicationRepository
		License() LicenseRepository
		Event() EventRepository
	}

	// PublicationRepository interface, defining publication operations
	PublicationRepository interface {
		ListAll() (*[]Publication, error)
		List(pageSize, pageNum int) (*[]Publication, error)
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
		List(pageSize, pageNum int) (*[]LicenseInfo, error)
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
)

// implementation of the Store interface
func (s *dbStore) Publication() PublicationRepository {
	return (*publicationStore)(s)
}

func (s *dbStore) License() LicenseRepository {
	return (*licenseStore)(s)
}

func (s *dbStore) Event() EventRepository {
	return (*eventStore)(s)
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

// DBSetup initializes the database
func DBSetup(dsn string) (Store, error) {
	var err error

	dialect, cnx := dbFromURI(dsn)
	if dialect == "error" {
		return nil, fmt.Errorf("incorrect database source name: %q", dsn)
	}

	var dialector gorm.Dialector
	// the use of time.Time fields for mysql requires parseTime
	if dialect == "mysql" && !strings.Contains(cnx, "parseTime") {
		return nil, fmt.Errorf("incomplete mysql database source name, parseTime required: %q", dsn)
	} else if dialect == "sqlite3" {
		dialector = sqlite.Open(cnx)
	}
	// Any constraint for other databases?

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

	db, err := gorm.Open(dialector, &gorm.Config{
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

	db.AutoMigrate(&Publication{}, &LicenseInfo{}, &Event{})

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
