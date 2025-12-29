package dto

type RegisterRequest struct {
	Name     string `json:"name" validate:"required,min=2,max=50"`
	Email    string `json:"email" validate:"required,email,max=255"`
	Password string `json:"password" validate:"required,min=8,max=72"` // Max 72 is a common bcrypt limit
}

type RegisterResponse struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// LoginRequest/Response for authentication
type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

// RefreshRequest/Response for token rotation
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

type RefreshResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"`
}

// PasswordChangeSendOTPRequest for initiating password change with OTP
type PasswordChangeSendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// PasswordChangeSendOTPResponse confirms OTP was sent
type PasswordChangeSendOTPResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}

// PasswordChangeRequest for completing password change with OTP verification
type PasswordChangeRequest struct {
	OldPassword string `json:"old_password" validate:"required,min=8,max=72"`
	NewPassword string `json:"new_password" validate:"required,min=8,max=72"`
	OTPCode     string `json:"otp_code" validate:"required,len=6"`
}

// PasswordChangeResponse for successful password change
type PasswordChangeResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type ResetPasswordRequest struct {
	Email       string `json:"email" validate:"required,email"`
	Code        string `json:"code" validate:"required,len=6"` // The OTP
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// ForgotPasswordSendOTPRequest initiates password reset with OTP
type ForgotPasswordSendOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
}

// ForgotPasswordSendOTPResponse confirms OTP was sent (always 200 OK, logs silently if email not found)
type ForgotPasswordSendOTPResponse struct {
	Message string `json:"message"`
}

// ResetPasswordWithOTPRequest completes password reset with OTP validation
type ResetPasswordWithOTPRequest struct {
	Email string `json:"email" validate:"required,email"`
	OTP   string `json:"otp" validate:"required,len=6"`
}

// ResetPasswordWithOTPResponse confirms password was reset
type ResetPasswordWithOTPResponse struct {
	Message string `json:"message"`
	Email   string `json:"email"`
}
