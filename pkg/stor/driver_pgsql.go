//go:build PGSQL

package stor

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	log.Println("PostgreSQL database")
	return postgres.Open(cnx)
}
