package models

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
	"time"
)

var DB *gorm.DB

const maxRetries = 5
const retryInterval = 5 * time.Second

func ConnectPostgresDatabase() {
	dsn := os.Getenv("DB_CONNECTION_STRING")
	if dsn == "" {
		log.Fatal("DB_CONNECTION_STRING não configurado")
	}
	var database *gorm.DB
	var err error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
		if err == nil {
			break
		}
		time.Sleep(retryInterval)
	}
	if err != nil {
		log.Fatalf("Falha ao conectar ao banco de dados após %d tentativas: %s", maxRetries, err)
	}
	if os.Getenv("ENV") != "production" {
		database.AutoMigrate(&Product{}, &Category{}, &CategoryProducts{}, &Additional{}, &AdditionalProducts{}, &OrderRecord{}, &OrderItem{}, &Delivery{})
	}
	DB = database
}
