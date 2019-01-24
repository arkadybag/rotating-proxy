package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"os"
)

var (
	host     = os.Getenv("host")
	port     = os.Getenv("port")
	user     = os.Getenv("user")
	dbname   = os.Getenv("dbname")
	password = os.Getenv("password")
)

func NewPostgreSQL() (db *gorm.DB, err error) {
	dbString := "host=%s port=%s user=%s dbname=%s password=%s"
	pgConf := fmt.Sprintf(dbString, host, port, user, dbname, password)

	connStr := fmt.Sprintf(pgConf)

	db, err = gorm.Open("postgres", connStr)
	if err != nil {
		return
	}
	return
}
