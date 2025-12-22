package repository

//
//import (
//	"time"
//
//	"mein-idaas/model"
//
//	"github.com/google/uuid"
//	"gorm.io/gorm"
//)
//
//// UserRepository defines DB operations for Users
//type UserRepository interface {
//	Create(user *model.User) error
//	GetByID(id uuid.UUID) (*model.User, error)
//	GetByEmail(email string) (*model.User, error)
//	Update(user *model.User) error
//	Delete(id uuid.UUID) error
//}
//
//// CredentialRepository defines DB operations for Credentials
//type CredentialRepository interface {
//	Create(cred *model.Credential) error
//	GetByID(id uuid.UUID) (*model.Credential, error)
//	GetByUserIDAndType(userID uuid.UUID, credType string) (*model.Credential, error)
//	Update(cred *model.Credential) error
//	Delete(id uuid.UUID) error
//}
//
//// RefreshTokenRepository defines DB operations for RefreshTokens
//type RefreshTokenRepository interface {
//	Create(rt *model.RefreshToken) error
//	GetByID(id uuid.UUID) (*model.RefreshToken, error)
//	GetByTokenHash(hash string) (*model.RefreshToken, error)
//	RevokeByHash(hash string) error
//	RevokeByID(id uuid.UUID) error
//	RevokeAllForUser(userID uuid.UUID) error // <--- NEW METHOD
//	Update(rt *model.RefreshToken) error
//	DeleteExpired() error
//	Delete(id uuid.UUID) error
//}
//
//// RoleRepository defines DB operations for Roles
//type RoleRepository interface {
//	GetByCode(code string) (*model.Role, error)
//	// You can add Create/Delete later if you build an Admin Dashboard
//}
//
//// ----------------- Postgres implementations -----------------
//
//type pgUserRepo struct {
//	db *gorm.DB
//}
//
//func NewUserRepository(db *gorm.DB) UserRepository {
//	return &pgUserRepo{db: db}
//}
//
//func (r *pgUserRepo) Create(user *model.User) error {
//	return r.db.Create(user).Error
//}
//
//func (r *pgUserRepo) GetByID(id uuid.UUID) (*model.User, error) {
//	var u model.User
//	// REMOVED .Preload("RefreshTokens") - This is the fix
//	if err := r.db.Preload("Roles").Preload("Credentials").First(&u, "id = ?", id).Error; err != nil {
//		return nil, err
//	}
//	return &u, nil
//}
//
//func (r *pgUserRepo) GetByEmail(email string) (*model.User, error) {
//	var u model.User
//	// FIXED: Added .Preload("Roles")
//	// This is required for the Login flow to bake roles into the initial JWT
//	if err := r.db.Preload("Roles").Preload("Credentials").Where("email = ?", email).First(&u).Error; err != nil {
//		return nil, err
//	}
//	return &u, nil
//}
//
//func (r *pgUserRepo) Update(user *model.User) error {
//	return r.db.Save(user).Error
//}
//
//func (r *pgUserRepo) Delete(id uuid.UUID) error {
//	return r.db.Delete(&model.User{}, "id = ?", id).Error
//}
//
//// ----------------- Credential repo -----------------
//
//type pgCredentialRepo struct {
//	db *gorm.DB
//}
//
//func NewCredentialRepository(db *gorm.DB) CredentialRepository {
//	return &pgCredentialRepo{db: db}
//}
//
//func (r *pgCredentialRepo) Create(cred *model.Credential) error {
//	return r.db.Create(cred).Error
//}
//
//func (r *pgCredentialRepo) GetByID(id uuid.UUID) (*model.Credential, error) {
//	var c model.Credential
//	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
//		return nil, err
//	}
//	return &c, nil
//}
//
//func (r *pgCredentialRepo) GetByUserIDAndType(userID uuid.UUID, credType string) (*model.Credential, error) {
//	var c model.Credential
//	if err := r.db.Where("user_id = ? AND type = ?", userID, credType).First(&c).Error; err != nil {
//		return nil, err
//	}
//	return &c, nil
//}
//
//func (r *pgCredentialRepo) Update(cred *model.Credential) error {
//	return r.db.Save(cred).Error
//}
//
//func (r *pgCredentialRepo) Delete(id uuid.UUID) error {
//	return r.db.Delete(&model.Credential{}, "id = ?", id).Error
//}
//
//// ----------------- RefreshToken repo -----------------
//
//type pgRefreshTokenRepo struct {
//	db *gorm.DB
//}
//
//func NewRefreshTokenRepository(db *gorm.DB) RefreshTokenRepository {
//	return &pgRefreshTokenRepo{db: db}
//}
//
//func (r *pgRefreshTokenRepo) Create(rt *model.RefreshToken) error {
//	return r.db.Create(rt).Error
//}
//
//func (r *pgRefreshTokenRepo) GetByID(id uuid.UUID) (*model.RefreshToken, error) {
//	var t model.RefreshToken
//	if err := r.db.First(&t, "id = ?", id).Error; err != nil {
//		return nil, err
//	}
//	return &t, nil
//}
//
//func (r *pgRefreshTokenRepo) GetByTokenHash(hash string) (*model.RefreshToken, error) {
//	var t model.RefreshToken
//	if err := r.db.Where("token_hash = ?", hash).First(&t).Error; err != nil {
//		return nil, err
//	}
//	return &t, nil
//}
//
//func (r *pgRefreshTokenRepo) RevokeByHash(hash string) error {
//	return r.db.Model(&model.RefreshToken{}).Where("token_hash = ?", hash).Update("revoked_at", time.Now()).Error
//}
//
//func (r *pgRefreshTokenRepo) RevokeByID(id uuid.UUID) error {
//	return r.db.Model(&model.RefreshToken{}).
//		Where("id = ?", id).
//		Update("revoked_at", time.Now()).Error
//}
//
//func (r *pgRefreshTokenRepo) DeleteExpired() error {
//	return r.db.Where("expires_at < ?", time.Now()).Delete(&model.RefreshToken{}).Error
//}
//
//func (r *pgRefreshTokenRepo) Update(rt *model.RefreshToken) error {
//	return r.db.Save(rt).Error
//}
//
//func (r *pgRefreshTokenRepo) RevokeAllForUser(userID uuid.UUID) error {
//	return r.db.Model(&model.RefreshToken{}).
//		Where("user_id = ?", userID).
//		Update("revoked_at", time.Now()).Error
//}
//
//func (r *pgRefreshTokenRepo) Delete(id uuid.UUID) error {
//	return r.db.Delete(&model.RefreshToken{}, "id = ?", id).Error
//}
//
//// ----------------- Role repo -----------------
//
//type pgRoleRepo struct {
//	db *gorm.DB
//}
//
//func NewRoleRepository(db *gorm.DB) RoleRepository {
//	return &pgRoleRepo{db: db}
//}
//
//func (r *pgRoleRepo) GetByCode(code string) (*model.Role, error) {
//	var role model.Role
//	// matches the s.roleRepo.GetByCode("user") call in AuthService
//	if err := r.db.Where("code = ?", code).First(&role).Error; err != nil {
//		return nil, err
//	}
//	return &role, nil
//}
