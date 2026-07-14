package models

import "time"

type SolicitationRecord struct {
	OrderID           string    `gorm:"primaryKey;type:varchar(255)"`
	DeliverymanID     int64     `gorm:"index"`
	Status            string    `gorm:"index"`
	DeliverymanStatus string    `gorm:"index"`
	Payload           string    `gorm:"type:jsonb"`
	OperationDate     time.Time `gorm:"autoUpdateTime"`
}
