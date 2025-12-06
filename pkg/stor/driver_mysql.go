//go:build MYSQL

package stor

import (
	log "github.com/sirupsen/logrus"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func GormDialector(cnx string) gorm.Dialector {
	log.Println("MySQL database")
	return mysql.Open(cnx)
}
