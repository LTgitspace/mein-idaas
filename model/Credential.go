package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Credential struct {
	ID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID uuid.UUID `gorm:"type:uuid;not null;index:idx_user_credential,unique"`
	//Type      string    `gorm:"size:50;not null"`   "Deprecated"
	Type      CredentialType `gorm:"size:50;not null;index:idx_user_credential,unique"`
	Value     string         `gorm:"type:text;not null"` // hashed password or encrypted API key
	Active    bool           `gorm:"default:true"`
	CreatedAt time.Time      `gorm:"autoCreateTime"`
	UpdatedAt time.Time      `gorm:"autoUpdateTime"`

	// Foreign Key
	User User `gorm:"foreignKey:UserID"`
}

func (c *Credential) BeforeCreate(tx *gorm.DB) (err error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return
}
