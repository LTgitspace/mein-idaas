package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey,uniqueIndex"`
	Name        string    `gorm:"size:50;not null,uniqueIndex"` // UI Display (e.g., "Super Administrator")
	Code        string    `gorm:"size:50;not null;uniqueIndex"` // The value sent in JWT (e.g., "super_admin")
	Description string    `gorm:"size:255"`
	IsSystem    bool      `gorm:"default:false"` // Protects critical roles from deletion
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (r *Role) BeforeCreate(_ *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
