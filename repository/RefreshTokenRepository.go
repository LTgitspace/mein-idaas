package repository

import (
	"time"

	"mein-idaas/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RefreshTokenRepository interface {
	Create(rt *model.RefreshToken) error
	GetByID(id uuid.UUID) (*model.RefreshToken, error)
	GetByTokenHash(hash string) (*model.RefreshToken, error)
	RevokeByHash(hash string) error
	RevokeByID(id uuid.UUID) error
	RevokeAllForUser(userID uuid.UUID) error
	Update(rt *model.RefreshToken) error
	DeleteExpired() error
	Delete(id uuid.UUID) error
}

type pgRefreshTokenRepo struct {
	db *gorm.DB
}

func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
	return &pgRefreshTokenRepo{db: db}
}

func (r *pgRefreshTokenRepo) Create(rt *model.RefreshToken) error {
	return r.db.Create(rt).Error
}

func (r *pgRefreshTokenRepo) GetByID(id uuid.UUID) (*model.RefreshToken, error) {
	var t model.RefreshToken
	if err := r.db.First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgRefreshTokenRepo) GetByTokenHash(hash string) (*model.RefreshToken, error) {
	var t model.RefreshToken
	if err := r.db.Where("token_hash = ?", hash).First(&t).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgRefreshTokenRepo) RevokeByHash(hash string) error {
	return r.db.Model(&model.RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", time.Now()).Error
}

func (r *pgRefreshTokenRepo) RevokeByID(id uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("id = ?", id).
		Update("revoked_at", time.Now()).Error
}

func (r *pgRefreshTokenRepo) DeleteExpired() error {
	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.RefreshToken{}).Error
}

func (r *pgRefreshTokenRepo) Update(rt *model.RefreshToken) error {
	return r.db.Save(rt).Error
}

func (r *pgRefreshTokenRepo) RevokeAllForUser(userID uuid.UUID) error {
	return r.db.Model(&model.RefreshToken{}).
		Where("user_id = ?", userID).
		Update("revoked_at", time.Now()).Error
}

func (r *pgRefreshTokenRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.RefreshToken{}, "id = ?", id).Error
}
