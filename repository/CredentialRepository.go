package repository

import (
	"mein-idaas/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type CredentialRepository interface {
	Create(cred *model.Credential) error
	GetByID(id uuid.UUID) (*model.Credential, error)
	GetByUserIDAndType(userID uuid.UUID, credType string) (*model.Credential, error)
	Update(cred *model.Credential) error
	Delete(id uuid.UUID) error
}

type pgCredentialRepo struct {
	db *gorm.DB
}

func NewCredentialRepository(db *gorm.DB) CredentialRepository {
	return &pgCredentialRepo{db: db}
}

func (r *pgCredentialRepo) Create(cred *model.Credential) error {
	return r.db.Create(cred).Error
}

func (r *pgCredentialRepo) GetByID(id uuid.UUID) (*model.Credential, error) {
	var c model.Credential
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *pgCredentialRepo) GetByUserIDAndType(userID uuid.UUID, credType string) (*model.Credential, error) {
	var c model.Credential
	if err := r.db.Where("user_id = ? AND type = ?", userID, credType).First(&c).Error; err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *pgCredentialRepo) Update(cred *model.Credential) error {
	return r.db.Save(cred).Error
}

func (r *pgCredentialRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.Credential{}, "id = ?", id).Error
}
