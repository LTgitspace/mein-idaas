package dto

// VerifyEmailRequest is the JSON payload sent to /auth/verify
type VerifyEmailRequest struct {
	Email string `json:"email" validate:"required,email"`
	Code  string `json:"code"  validate:"required,len=6"` // Enforce exact 6 digits
}

// ResendOTPRequest is optional, but useful for a "Resend Code" button
type ResendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}
