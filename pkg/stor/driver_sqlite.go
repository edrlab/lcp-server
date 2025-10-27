//go:build !PGSQL && !MYSQL

package stor

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	println("SQLite database")
	return sqlite.Open(cnx)
}
