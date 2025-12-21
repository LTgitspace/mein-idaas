package model

import (
	"time"

	"github.com/google/uuid"
)

type Credential struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index"`
	Type      string    `gorm:"size:50;not null"`   // "password", "api_key"
	Value     string    `gorm:"type:text;not null"` // hashed password or encrypted API key
	Active    bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`

	// Foreign Key
	User User `gorm:"foreignKey:UserID"`
}
