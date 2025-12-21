package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID   uuid.UUID `gorm:"type:uuid;primaryKey;uniqueIndex"`
	Name string    `gorm:"size:50;not null;uniqueIndex"`

	Code        string    `gorm:"size:50;not null;uniqueIndex"`
	Description string    `gorm:"size:255"`
	IsSystem    bool      `gorm:"default:false"`
	CreatedAt   time.Time `gorm:"autoCreateTime"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

func (r *Role) BeforeCreate(_ *gorm.DB) (err error) {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return
}
