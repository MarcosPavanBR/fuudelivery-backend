package models

type AdditionalProducts struct {
	AdditionalID uint
	ProductID    uint
	Additional   Additional `gorm:"foreignKey:AdditionalID"`
	Product      Product    `gorm:"foreignKey:ProductID"`
}
