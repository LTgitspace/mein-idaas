package repository

import (
	"mein-idaas/model"

	"gorm.io/gorm"
)

type RoleRepository interface {
	GetByCode(code string) (*model.Role, error)
}

type pgRoleRepo struct {
	db *gorm.DB
}

func NewRoleRepository(db *gorm.DB) RoleRepository {
	return &pgRoleRepo{db: db}
}

func (r *pgRoleRepo) GetByCode(code string) (*model.Role, error) {
	var role model.Role
	if err := r.db.Where("code = ?", code).First(&role).Error; err != nil {
		return nil, err
	}
	return &role, nil
}
