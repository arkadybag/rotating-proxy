package main

import (
	"fmt"
	"os"

	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func NewPostgreSQL() (*gorm.DB, error) {
	host := os.Getenv("host")
	port := os.Getenv("port")
	user := os.Getenv("user")
	dbname := os.Getenv("dbname")
	password := os.Getenv("password")

	dbString := "host=%s port=%s user=%s dbname=%s password=%s"
	pgConf := fmt.Sprintf(dbString, host, port, user, dbname, password)

	return gorm.Open("postgres", pgConf)
}
