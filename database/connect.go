package database

import (
	"fmt"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
)

func Connect() *gorm.DB {
	db, err := gorm.Open("mysql", DBuser+":"+DBpass+"@/chat?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		fmt.Println("Error Connecting to DB")
		return nil
	}
	return db
}
