//go:build PGSQL

package stor

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	println("PostgreSQL database")
	return postgres.Open(cnx)
}
