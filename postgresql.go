package main

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
)

func NewPostgreSQL() (db *gorm.DB, err error) {
	connStr := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s",
		"89.179.126.31", "5432",
		"proxy", "proxy", "ghbdtn",
	)

	db, err = gorm.Open("postgres", connStr)
	if err != nil {
		return
	}
	return
}
