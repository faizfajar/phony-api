package database

import (
	"fmt"
	"log"

	"github.com/faizfajar/phony-api/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Connect(dsn string) {
	var err error
	DB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}

	fmt.Println("Database Connected!")

	// Auto-Migrate
	err = DB.AutoMigrate(&model.Endpoint{}, &model.Response{}, &model.APIMetric{})
	if err != nil {
		log.Fatal("Migration failed: ", err)
	}
}
