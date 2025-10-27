//go:build MYSQL

package stor

import (
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	println("MySQL database")
	return mysql.Open(cnx)
}
