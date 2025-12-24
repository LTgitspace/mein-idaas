package service

import (
	"errors"
	"log"

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
		if err := s.emailService.SendOTP(email, code); err != nil {
			log.Printf("Failed to send OTP to %s: %v", email, err)
			return
		}
		log.Printf("OTP sent successfully to %s", email)
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
		return errors.New("invalid verification code")
	}

	// 3. Cleanup (Prevent replay attacks)
	_ = s.repo.Delete(userID)

	return nil
}
