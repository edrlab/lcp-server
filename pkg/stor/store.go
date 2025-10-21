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

	// the use of time.Time fields for mysql requires parseTime
	if dialect == "mysql" && !strings.Contains(cnx, "parseTime") {
		return nil, fmt.Errorf("incomplete mysql database source name, parseTime required: %q", dsn)
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

	// Create indexes to optimize dashboard queries
	err = createDashboardIndexes(db)
	if err != nil {
		log.Printf("Failed creating dashboard indexes: %v", err)
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

// createDashboardIndexes creates necessary indexes to optimize dashboard queries
func createDashboardIndexes(db *gorm.DB) error {
	// Index on created_at for period queries
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_license_info_created_at ON license_infos(created_at)").Error; err != nil {
		return err
	}
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_publication_created_at ON publications(created_at)").Error; err != nil {
		return err
	}

	// Index on device_count for excessive sharing queries
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_license_info_device_count ON license_infos(device_count)").Error; err != nil {
		return err
	}

	// Index on content_type for publication type queries
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_publication_content_type ON publications(content_type)").Error; err != nil {
		return err
	}

	// Index on publication_id for JOIN queries in GetOversharedLicenses
	if err := db.Exec("CREATE INDEX IF NOT EXISTS idx_license_info_publication_id ON license_infos(publication_id)").Error; err != nil {
		return err
	}

	// Note: Indexes on status and user_id already exist in the LicenseInfo model via gorm tags
	return nil
}
