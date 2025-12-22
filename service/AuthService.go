package service

import (
	"errors"
	"time"

	"mein-idaas/dto"
	"mein-idaas/model"
	"mein-idaas/repository"
	"mein-idaas/util"
)

type AuthService struct {
	userRepo       repository.UserRepository
	credentialRepo repository.CredentialRepository
	refreshRepo    repository.RefreshTokenRepository
	roleRepo       repository.RoleRepository // <--- ADDED
}

// NewAuthService now requires RoleRepository
func NewAuthService(
	u repository.UserRepository,
	c repository.CredentialRepository,
	r repository.RefreshTokenRepository,
	role repository.RoleRepository, // <--- ADDED argument
) *AuthService {
	return &AuthService{
		userRepo:       u,
		credentialRepo: c,
		refreshRepo:    r,
		roleRepo:       role, // <--- ADDED assignment
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

	rt := &model.RefreshToken{
		ID:        pair.RefreshID,
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
		ClientIP:  clientIP,
		UserAgent: userAgent,
	}
	if err := s.refreshRepo.Create(rt); err != nil {
		return nil, err
	}

	return &dto.LoginResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: 15 * 60}, nil
}

// Refresh rotates refresh tokens and issues a new access token
//func (s *AuthService) Refresh(req *dto.RefreshRequest, clientIP, userAgent string) (*dto.RefreshResponse, error) {
//	userIDFromToken, refreshID, err := util.ParseRefreshToken(req.RefreshToken)
//	if err != nil {
//		return nil, errors.New("invalid refresh token")
//	}
//
//	existing, err := s.refreshRepo.GetByID(refreshID)
//	if err != nil {
//		return nil, errors.New("invalid or unknown refresh token")
//	}
//
//	if existing.UserID != userIDFromToken {
//		return nil, errors.New("refresh token user mismatch")
//	}
//
//	if existing.TokenHash != util.HashToken(req.RefreshToken) {
//		return nil, errors.New("refresh token mismatch")
//	}
//
//	if !existing.IsValid() {
//		return nil, errors.New("refresh token expired or revoked")
//	}
//
//	// Fetch User to get LATEST roles
//	user, err := s.userRepo.GetByID(existing.UserID)
//	if err != nil {
//		return nil, errors.New("user not found")
//	}
//
//	var roleCodes []string
//	for _, r := range user.Roles {
//		roleCodes = append(roleCodes, r.Code)
//	}
//
//	// Generate new tokens
//	pair, err := util.GenerateTokens(existing.UserID, roleCodes)
//	if err != nil {
//		return nil, err
//	}
//
//	if err := s.refreshRepo.RevokeByID(existing.ID); err != nil {
//		// log error
//	}
//
//	newHash := util.HashToken(pair.RefreshToken)
//	newRT := &model.RefreshToken{
//		ID:        pair.RefreshID,
//		UserID:    existing.UserID,
//		TokenHash: newHash,
//		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
//		ClientIP:  clientIP,
//		UserAgent: userAgent,
//	}
//	if err := s.refreshRepo.Create(newRT); err != nil {
//		return nil, err
//	}
//
//	return &dto.RefreshResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: 15 * 60}, nil
//}

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
	} // Manual logout

	// ---------------------------------------------------------
	// 4. GRACE PERIOD & REUSE DETECTION LOGIC
	// ---------------------------------------------------------

	if existing.ReplacedAt != nil {
		duration := time.Since(*existing.ReplacedAt)

		// CASE A: Theft Detected (Replay attack after 10s)
		if duration > 10*time.Second {
			// "Nuclear Option": Revoke ALL tokens for this user because their family is compromised
			// You need to add this method to your Repo
			// s.refreshRepo.RevokeAllForUser(existing.UserID)
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

		// 3. Generate ONLY a new Access Token (Use a helper or ignore the returned RT)
		// We DO NOT want a new Refresh ID. We want to keep the chain A -> B.
		newAccessToken, err := util.GenerateAccessTokenOnly(user.ID, roleCodes)
		if err != nil {
			return nil, err
		}

		// 4. Re-sign the EXISTING child token ID so the client gets a valid JWT string
		// This ensures the client gets the same Refresh Token ID that is already in the DB
		refreshTokenString, err := util.SignRefreshToken(childToken.ID, childToken.UserID)
		if err != nil {
			return nil, err
		}

		// 5. RETURN IMMEDIATELY. DO NOT UPDATE DB.
		// The DB is already correct (A -> ReplacedBy -> B). We just resend the info.
		return &dto.RefreshResponse{
			AccessToken:  newAccessToken,
			RefreshToken: refreshTokenString,
			ExpiresIn:    900,
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

	// Save the NEW Token
	newHash := util.HashToken(pair.RefreshToken)
	newRT := &model.RefreshToken{
		ID:        pair.RefreshID,
		UserID:    existing.UserID,
		TokenHash: newHash,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
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
		// If we fail to mark the old one as replaced, we have a security issue.
		// Ideally, delete the 'newRT' we just created to prevent orphans.
		s.refreshRepo.Delete(newRT.ID)
		return nil, errors.New("failed to rotate token")
	}

	return &dto.RefreshResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: 900}, nil
}
