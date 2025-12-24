package service

import (
	"mein-idaas/repository"
	"mein-idaas/util"
	"time"
)

type VerificationService struct {
	repo         repository.VerificationRepository
	emailService *EmailService
}

// NewVerificationService injects dependencies
func NewVerificationService(repo repository.VerificationRepository, emailService *EmailService) *VerificationService {
	return &VerificationService{
		repo:         repo,
		emailService: emailService,
	}
}

// SendVerificationCode orchestrates the entire flow
func (s *VerificationService) SendVerificationCode(userID string, email string) error {
	// 1. Generate 6-digit Code
	code := util.GenerateRandomDigits(6)

	// 2. Save to Repository (TTL 5 minutes)
	// We use userID as key so one user can't spam multiple codes easily
	err := s.repo.Save(userID, code, 5*time.Minute)
	if err != nil {
		return err
	}

	// 3. Send Email (Run in background so API is fast)
	go func() {
		_ = s.emailService.SendOTP(email, code)
		// in production, you might want to log the error here if it fails
	}()

	return nil
}

// VerifyCode checks if the code is correct
func (s *VerificationService) VerifyCode(userID string, inputCode string) error {
	// 1. Get from Repo
	savedCode, err := s.repo.Get(userID)
	if err != nil {
		return err // "code expired" or "not found"
	}

	// 2. Compare
	if savedCode != inputCode {
		// optional: decrease retry count here to prevent brute force
		return err // or generic "invalid code"
	}

	// 3. Cleanup (Prevent replay attacks)
	_ = s.repo.Delete(userID)

	return nil
}
