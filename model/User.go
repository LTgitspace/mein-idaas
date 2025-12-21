package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID              uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name            string    `gorm:"size:50;not null"`
	IsEmailVerified bool      `gorm:"default:false"` // Critical for Identity Systems
	Email           string    `gorm:"size:255;not null;uniqueIndex"`
	CreatedAt       time.Time `gorm:"autoCreateTime"`
	UpdatedAt       time.Time `gorm:"autoUpdateTime"`

	Credentials   []Credential   `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	RefreshTokens []RefreshToken `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE;"`
	Roles         []Role         `gorm:"many2many:user_roles;constraint:OnDelete:CASCADE;"`
}

type JSONB map[string]interface{}

func (b *User) BeforeCreate(_ *gorm.DB) (err error) {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return
}
