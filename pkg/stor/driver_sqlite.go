//go:build !PGSQL

package stor

import (
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {

	return sqlite.Open(cnx)
}
