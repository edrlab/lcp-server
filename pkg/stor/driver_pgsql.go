//go:build PGSQL

package stor

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	log.Println("Using PostgreSQL")
	return postgres.Open(cnx)
}
