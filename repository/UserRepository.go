package repository

import (
	"mein-idaas/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uuid.UUID) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uuid.UUID) error
	GetDB() *gorm.DB
}

type pgUserRepo struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &pgUserRepo{db: db}
}

func (r *pgUserRepo) Create(user *model.User) error {
	return r.db.Create(user).Error
}

func (r *pgUserRepo) GetByID(id uuid.UUID) (*model.User, error) {
	var u model.User
	// Fetches Roles and Credentials to ensure the user object is complete
	if err := r.db.Preload("Roles").Preload("Credentials").First(&u, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgUserRepo) GetByEmail(email string) (*model.User, error) {
	var u model.User
	// Preload Roles here so they are available for JWT generation during Login
	if err := r.db.Preload("Roles").Preload("Credentials").Where("email = ?", email).First(&u).Error; err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgUserRepo) Update(user *model.User) error {
	return r.db.Save(user).Error
}

func (r *pgUserRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.User{}, "id = ?", id).Error
}

func (r *pgUserRepo) GetDB() *gorm.DB {
	return r.db
}
