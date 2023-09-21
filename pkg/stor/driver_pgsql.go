//go:build PGSQL

package stor

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {

	return postgres.Open(cnx)
}
