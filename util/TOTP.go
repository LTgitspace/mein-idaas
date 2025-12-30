package util

import (
	"bytes"
	"fmt"
	"image/png"
	"os"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
)

// GenerateTOTPSecret creates a new TOTP secret for MFA
func GenerateTOTPSecret(email string) (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      os.Getenv("APP_NAME"),
		AccountName: email,
	})
	if err != nil {
		return "", fmt.Errorf("failed to generate TOTP secret: %w", err)
	}
	return key.Secret(), nil
}

// GetTOTPQRCode generates a QR code image for TOTP enrollment
func GetTOTPQRCode(secret, email string) ([]byte, error) {
	key, err := otp.NewKeyFromURL(fmt.Sprintf("otpauth://totp/%s?secret=%s&issuer=%s",
		email, secret, os.Getenv("APP_NAME")))
	if err != nil {
		return nil, fmt.Errorf("failed to create OTP key: %w", err)
	}

	img, err := key.Image(256, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Convert image to PNG bytes
	buf := new(bytes.Buffer)
	if err := png.Encode(buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode QR code to PNG: %w", err)
	}

	return buf.Bytes(), nil
}

// VerifyTOTP validates a TOTP token
func VerifyTOTP(secret, token string) bool {
	return totp.Validate(token, secret)
}
