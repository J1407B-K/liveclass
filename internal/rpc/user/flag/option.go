package flag

import (
	"flag"
	"gorm.io/gorm"
)

type Option struct {
	DB bool
}

func Parse() Option {
	db := flag.Bool("db", false, "parse db")
	flag.Parse()

	return Option{
		DB: *db,
	}
}

func DBOption(db *gorm.DB, o Option) bool {
	if o.DB {
		MysqlAutoMigrate(db)
		return true
	}
	return false
}
