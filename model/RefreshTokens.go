package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshToken struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID  `gorm:"type:uuid;not null;index"`
	TokenHash         string     `gorm:"type:text;not null;uniqueIndex"` // Hash of actual token
	ClientIP          string     `gorm:"size:45"`                        // IPv6 support
	UserAgent         string     `gorm:"type:text"`
	ExpiresAt         time.Time  `gorm:"not null;index"`
	ReplacedAt        *time.Time // When it was rotated
	ReplacedByTokenID *uuid.UUID // Points to the new child token
	RevokedAt         *time.Time `gorm:"index"` // NULL if not revoked
	CreatedAt         time.Time  `gorm:"autoCreateTime"`

	// Foreign Key
	User User `gorm:"foreignKey:UserID"`
}

func (rt *RefreshToken) BeforeCreate(_ *gorm.DB) error {
	if rt.ID == uuid.Nil {
		rt.ID = uuid.New()
	}
	return nil
}

// IsValid checks if refresh token is still usable
func (rt *RefreshToken) IsValid() bool {
	return time.Now().Before(rt.ExpiresAt) && rt.RevokedAt == nil
}
