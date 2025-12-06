//go:build !PGSQL && !MYSQL

package stor

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	log.Println("SQLite database")
	return sqlite.Open(cnx)
}
