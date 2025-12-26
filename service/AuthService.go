package service

import (
	"errors"
	"log"
	"os"
	_ "strconv"
	"time"

	"mein-idaas/dto"
	"mein-idaas/model"
	"mein-idaas/repository"
	"mein-idaas/util"

	"github.com/google/uuid"
)

type AuthService struct {
	userRepo        repository.UserRepository
	credentialRepo  repository.CredentialRepository
	refreshRepo     repository.RefreshTokenRepository
	roleRepo        repository.RoleRepository
	verificationSvc *VerificationService
}

// NewAuthService now requires RoleRepository and VerificationService
func NewAuthService(
	u repository.UserRepository,
	c repository.CredentialRepository,
	r repository.RefreshTokenRepository,
	role repository.RoleRepository,
	verification *VerificationService,
) *AuthService {
	return &AuthService{
		userRepo:        u,
		credentialRepo:  c,
		refreshRepo:     r,
		roleRepo:        role,
		verificationSvc: verification,
	}
}

// Register creates a new user, assigns default role, and creates credentials
func (s *AuthService) Register(req *dto.RegisterRequest) (*dto.RegisterResponse, error) {
	// 1. Start a Transaction (All or Nothing)
	tx := s.userRepo.GetDB().Begin()

	// Safety: Rollback if panic occurs or if we forget to commit
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 2. Prepare User
	user := &model.User{
		Name:  req.Name,
		Email: req.Email,
	}

	// 3. Attach Role
	defaultRole, err := s.roleRepo.GetByCode("user")
	if err != nil {
		tx.Rollback()
		return nil, errors.New("system error: default role not found")
	}
	user.Roles = append(user.Roles, *defaultRole)

	// 🛡️ CRITICAL SAFETY: Force Credentials to nil to prevent "Double Save"
	user.Credentials = nil

	// 4. Create User (USING 'tx', not 's.userRepo')
	if err := tx.Create(user).Error; err != nil {
		tx.Rollback()
		if util.IsDuplicateKeyError(err) {
			return nil, errors.New("email already in use")
		}
		return nil, err
	}

	// 5. Hash Password
	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// 6. Create Credential (USING 'tx')
	cred := &model.Credential{
		UserID: user.ID,
		Type:   model.CredTypePassword, // Make sure this matches your Enum
		Value:  hashed,
	}

	if err := tx.Debug().Create(cred).Error; err != nil {
		tx.Rollback()
		// This will print the exact SQL error to your API response
		return nil, errors.New("SQL ERROR: " + err.Error())
	}

	// 7. Commit (Save everything permanently)
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// 8. Trigger verification email asynchronously (log failures)
	if s.verificationSvc != nil {
		if err := s.verificationSvc.SendVerificationCode(user.ID.String(), user.Email); err != nil {
			log.Printf("failed to initiate verification email for %s: %v", user.Email, err)
		} else {
			log.Printf("verification email initiated for %s", user.Email)
		}
	} else {
		log.Printf("no verification service configured; skipping verification email for %s", user.Email)
	}

	return &dto.RegisterResponse{ID: user.ID.String(), Name: user.Name, Email: user.Email}, nil
}

// Login validates credentials and returns a token pair
func (s *AuthService) Login(req *dto.LoginRequest, clientIP, userAgent string) (*dto.LoginResponse, error) {
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, errors.New("invalid credentials")
	}

	var pwCred *model.Credential
	for _, c := range user.Credentials {
		if c.Type == model.CredTypePassword {
			pwCred = &c
			break
		}
	}
	if pwCred == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := util.ComparePassword(pwCred.Value, req.Password); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Check if email is verified
	if !user.IsEmailVerified {
		// Send verification email asynchronously (log failures)
		if s.verificationSvc != nil {
			if err := s.verificationSvc.SendVerificationCode(user.ID.String(), user.Email); err != nil {
				log.Printf("failed to send verification email for %s: %v", user.Email, err)
			} else {
				log.Printf("verification email sent for unverified user %s", user.Email)
			}
		}
		return nil, errors.New("email not verified")
	}

	// Extract Roles for Token
	var roleCodes []string
	for _, r := range user.Roles {
		roleCodes = append(roleCodes, r.Code)
	}

	// Generate Tokens with Roles
	pair, err := util.GenerateTokens(user.ID, roleCodes)
	if err != nil {
		return nil, err
	}

	hash := util.HashToken(pair.RefreshToken)

	// Get refresh TTL from env (default 168h = 7 days)
	refreshTTLStr := os.Getenv("JWT_REFRESH_TTL")
	if refreshTTLStr == "" {
		refreshTTLStr = "168h"
	}
	refreshTTL, _ := time.ParseDuration(refreshTTLStr)

	rt := &model.RefreshToken{
		ID:        pair.RefreshID,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(refreshTTL),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}
	if err := s.refreshRepo.Create(rt); err != nil {
		return nil, err
	}

	// Get access token TTL in seconds for response
	accessTTLStr := os.Getenv("JWT_ACCESS_TTL")
	if accessTTLStr == "" {
		accessTTLStr = "15m"
	}
	accessTTL, _ := time.ParseDuration(accessTTLStr)
	expiresIn := int(accessTTL.Seconds())

	return &dto.LoginResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: expiresIn}, nil
}

// Refresh rotates refresh tokens and issues a new access token
func (s *AuthService) Refresh(req *dto.RefreshRequest, clientIP, userAgent string) (*dto.RefreshResponse, error) {
	// 1. Parse & Validate basic structure
	userIDFromToken, refreshID, err := util.ParseRefreshToken(req.RefreshToken)
	if err != nil {
		return nil, errors.New("invalid refresh token")
	}

	// 2. Load Token from DB
	existing, err := s.refreshRepo.GetByID(refreshID)
	if err != nil {
		return nil, errors.New("invalid or unknown refresh token")
	}

	// 3. Security Checks
	if existing.UserID != userIDFromToken {
		return nil, errors.New("user mismatch")
	}
	if existing.RevokedAt != nil {
		return nil, errors.New("token was revoked")
	}

	// ---------------------------------------------------------
	// 4. GRACE PERIOD & REUSE DETECTION LOGIC
	// ---------------------------------------------------------

	if existing.ReplacedAt != nil {
		duration := time.Since(*existing.ReplacedAt)

		// Get grace period from env (default 10s)
		gracePeriodStr := os.Getenv("REFRESH_GRACE_PERIOD")
		if gracePeriodStr == "" {
			gracePeriodStr = "10s"
		}
		gracePeriod, _ := time.ParseDuration(gracePeriodStr)

		// CASE A: Theft Detected (Replay attack after grace period)
		if duration > gracePeriod {
			return nil, errors.New("refresh token reuse detected: account locked for security")
		}

		// CASE B: Grace Period (Concurrency retry)
		if existing.ReplacedByTokenID == nil {
			return nil, errors.New("system inconsistency: replaced timestamp set but no replacement ID")
		}

		// 1. Find the token that ALREADY replaced this one
		childToken, err := s.refreshRepo.GetByID(*existing.ReplacedByTokenID)
		if err != nil {
			return nil, errors.New("child token not found")
		}

		// 2. Fetch User for Roles
		user, err := s.userRepo.GetByID(existing.UserID)
		if err != nil {
			return nil, errors.New("failed to fetch user")
		}
		var roleCodes []string
		for _, r := range user.Roles {
			roleCodes = append(roleCodes, r.Code)
		}

		// 3. Generate ONLY a new Access Token
		newAccessToken, err := util.GenerateAccessTokenOnly(user.ID, roleCodes)
		if err != nil {
			return nil, err
		}

		// 4. Re-sign the EXISTING child token ID
		refreshTokenString, err := util.SignRefreshToken(childToken.ID, childToken.UserID)
		if err != nil {
			return nil, err
		}

		// 5. Get access token TTL in seconds for response
		accessTTLStr := os.Getenv("JWT_ACCESS_TTL")
		if accessTTLStr == "" {
			accessTTLStr = "15m"
		}
		accessTTL, _ := time.ParseDuration(accessTTLStr)
		expiresIn := int(accessTTL.Seconds())

		return &dto.RefreshResponse{
			AccessToken:  newAccessToken,
			RefreshToken: refreshTokenString,
			ExpiresIn:    expiresIn,
		}, nil
	}

	// ---------------------------------------------------------
	// 5. NORMAL ROTATION (First time using this token)
	// ---------------------------------------------------------

	// Fetch User Roles
	user, err := s.userRepo.GetByID(existing.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	var roleCodes []string
	for _, r := range user.Roles {
		roleCodes = append(roleCodes, r.Code)
	}

	// Generate NEW Pair
	pair, err := util.GenerateTokens(existing.UserID, roleCodes)
	if err != nil {
		return nil, err
	}

	// Get refresh TTL from env (default 168h = 7 days)
	refreshTTLStr := os.Getenv("JWT_REFRESH_TTL")
	if refreshTTLStr == "" {
		refreshTTLStr = "168h"
	}
	refreshTTL, _ := time.ParseDuration(refreshTTLStr)

	// Save the NEW Token
	newHash := util.HashToken(pair.RefreshToken)
	newRT := &model.RefreshToken{
		ID:        pair.RefreshID,
		UserID:    existing.UserID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(refreshTTL),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}
	if err := s.refreshRepo.Create(newRT); err != nil {
		return nil, err
	}

	// Mark OLD Token as Replaced (Link it to the new one)
	now := time.Now()
	existing.ReplacedAt = &now
	existing.ReplacedByTokenID = &pair.RefreshID
	if err := s.refreshRepo.Update(existing); err != nil {
		_ = s.refreshRepo.Delete(newRT.ID)
		return nil, errors.New("failed to rotate token")
	}

	// Get access token TTL in seconds for response
	accessTTLStr := os.Getenv("JWT_ACCESS_TTL")
	if accessTTLStr == "" {
		accessTTLStr = "15m"
	}
	accessTTL, _ := time.ParseDuration(accessTTLStr)
	expiresIn := int(accessTTL.Seconds())

	return &dto.RefreshResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: expiresIn}, nil
}

// GetUserByID retrieves a user by ID with their roles and credentials
func (s *AuthService) GetUserByID(userID string) (*model.User, error) {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, errors.New("invalid user ID format")
	}
	return s.userRepo.GetByID(uid)
}

// GetUserByEmail retrieves a user by email with their roles and credentials
func (s *AuthService) GetUserByEmail(email string) (*model.User, error) {
	return s.userRepo.GetByEmail(email)
}

// StoreRefreshToken stores a refresh token in the database
func (s *AuthService) StoreRefreshToken(tokenID string, userID interface{}, tokenHash string, ttl time.Duration, clientIP, userAgent string) error {
	// Parse tokenID as UUID
	uuidID := uuid.MustParse(tokenID)

	// Ensure userID is uuid.UUID
	var uuidUserID uuid.UUID
	switch v := userID.(type) {
	case uuid.UUID:
		uuidUserID = v
	case string:
		uuidUserID = uuid.MustParse(v)
	default:
		return errors.New("invalid user ID type")
	}

	rt := &model.RefreshToken{
		ID:        uuidID,
		UserID:    uuidUserID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(ttl),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}
	return s.refreshRepo.Create(rt)
}

// MarkEmailVerified sets IsEmailVerified = true for the specified user
func (s *AuthService) MarkEmailVerified(userID string) error {
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}

	if user.IsEmailVerified {
		return nil
	}

	user.IsEmailVerified = true
	if err := s.userRepo.Update(user); err != nil {
		return err
	}
	return nil
}

// SendPasswordChangeOTP sends an OTP to the user's email for password change
func (s *AuthService) SendPasswordChangeOTP(email string) error {
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return errors.New("user not found")
	}

	// Send OTP via verification service
	if s.verificationSvc != nil {
		if err := s.verificationSvc.SendVerificationCode(user.ID.String(), user.Email); err != nil {
			log.Printf("failed to send password change OTP to %s: %v", user.Email, err)
			return err
		}
		log.Printf("password change OTP sent successfully to %s", user.Email)
		return nil
	}
	return errors.New("verification service not configured")
}

// SendPasswordChangeOTPByUserID sends an OTP to the user's email using their user ID
func (s *AuthService) SendPasswordChangeOTPByUserID(userID string) (string, error) {
	// 1. Parse userID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return "", errors.New("invalid user ID format")
	}

	// 2. Get user by ID
	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return "", errors.New("user not found")
	}

	// 3. Send OTP via verification service
	if s.verificationSvc != nil {
		if err := s.verificationSvc.SendPasswordChangeCode(user.ID.String(), user.Email); err != nil {
			log.Printf("failed to send password change OTP to %s: %v", user.Email, err)
			return "", err
		}
		log.Printf("password change OTP sent successfully to %s", user.Email)
		return user.Email, nil
	}
	return "", errors.New("verification service not configured")
}

// ChangePassword changes the user's password after OTP verification
func (s *AuthService) ChangePassword(userID string, oldPassword string, newPassword string, otpCode string) error {
	// 1. Parse userID
	uid, err := uuid.Parse(userID)
	if err != nil {
		return errors.New("invalid user ID format")
	}

	// 2. Verify OTP
	if s.verificationSvc != nil {
		if err := s.verificationSvc.VerifyCode(userID, otpCode); err != nil {
			return err
		}
	} else {
		return errors.New("verification service not configured")
	}

	// 3. Get user with credentials
	user, err := s.userRepo.GetByID(uid)
	if err != nil {
		return err
	}

	// 4. Find existing password credential
	var pwCred *model.Credential
	for i, c := range user.Credentials {
		if c.Type == model.CredTypePassword {
			pwCred = &user.Credentials[i]
			break
		}
	}
	if pwCred == nil {
		return errors.New("password credential not found")
	}

	// 5. Verify old password
	if err := util.ComparePassword(pwCred.Value, oldPassword); err != nil {
		return errors.New("invalid old password")
	}

	// 6. Hash new password
	hashedNewPassword, err := util.HashPassword(newPassword)
	if err != nil {
		return err
	}

	// 7. Update credential
	pwCred.Value = hashedNewPassword
	if err := s.credentialRepo.Update(pwCred); err != nil {
		return err
	}

	log.Printf("password changed successfully for user %s", user.Email)
	return nil
}
