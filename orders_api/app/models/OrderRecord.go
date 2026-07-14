package models

import "time"

type OrderRecord struct {
	ID              string    `gorm:"primaryKey;type:varchar(255)"`
	EstablishmentID int64     `gorm:"index"`
	Phone           string    `gorm:"index"`
	Status          string    `gorm:"index"`
	Payload         string    `gorm:"type:jsonb"`
	LastModified    time.Time `gorm:"autoUpdateTime"`
}
