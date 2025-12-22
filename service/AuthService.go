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
	// 1. Prepare User Object
	user := &model.User{
		Name:  req.Name,
		Email: req.Email,
	}

	// 2. Fetch Default Role (e.g., "user")
	// This ensures every new registration gets the basic permissions
	defaultRole, err := s.roleRepo.GetByCode("user")
	if err != nil {
		// If "user" role is missing, registration should probably fail
		// so you don't have users with 0 permissions.
		return nil, errors.New("system error: default role not found")
	}

	// Attach the role. GORM will handle the join table insertion when we Create(user).
	user.Roles = append(user.Roles, *defaultRole)

	// 3. Persist User (and the relationship to the Role)
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	// 4. Hash Password
	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	// 5. Create Password Credential
	cred := &model.Credential{
		UserID: user.ID,
		Type:   "password",
		Value:  hashed,
	}
	if err := s.credentialRepo.Create(cred); err != nil {
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
		if c.Type == "password" {
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

		// CASE B: Grace Period
		if existing.ReplacedByTokenID == nil {
			return nil, errors.New("system inconsistency: replaced timestamp set but no replacement ID")
		}

		childToken, err := s.refreshRepo.GetByID(*existing.ReplacedByTokenID)
		if err != nil {
			return nil, errors.New("child token not found")
		}

		// 1. FIX: Check Error (Don't ignore it with _)
		user, err := s.userRepo.GetByID(existing.UserID)
		if err != nil {
			return nil, errors.New("failed to fetch user")
		}

		var roleCodes []string
		for _, r := range user.Roles {
			roleCodes = append(roleCodes, r.Code)
		}

		// 2. FIX: Check Error (Don't ignore it with _)
		pair, err := util.GenerateTokens(existing.UserID, roleCodes)
		if err != nil {
			return nil, err // Return the actual error instead of crashing
		}

		rtString, err := util.SignRefreshToken(childToken.ID, childToken.UserID)
		if err != nil {
			return nil, err
		}

		now := time.Now()
		existing.ReplacedAt = &now
		existing.ReplacedByTokenID = &pair.RefreshID

		// 🔴 CRITICAL FIX: Check this error!
		// If this fails, the token remains "fresh" and can be reused forever.
		if err := s.refreshRepo.Update(existing); err != nil {
			return nil, errors.New("critical: failed to mark token as used")
		}

		return &dto.RefreshResponse{
			AccessToken:  pair.AccessToken,
			RefreshToken: rtString,
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
	s.refreshRepo.Update(existing) // Save changes

	return &dto.RefreshResponse{AccessToken: pair.AccessToken, RefreshToken: pair.RefreshToken, ExpiresIn: 900}, nil
}
