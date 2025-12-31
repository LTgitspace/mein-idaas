package service

import (
	"crypto/tls"
	"fmt"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
)

type EmailService struct {
	dialer *gomail.Dialer
	sender string
}

func NewEmailService() *EmailService {
	// Read from .env
	host := os.Getenv("SMTP_HOST")
	portStr := os.Getenv("SMTP_PORT")
	user := os.Getenv("SMTP_USER")
	pass := os.Getenv("SMTP_PASS")
	sender := os.Getenv("SMTP_SENDER_NAME")

	port, _ := strconv.Atoi(portStr)

	dialer := gomail.NewDialer(host, port, user, pass)

	// TLS configuration: Allow self-signed certs in dev, strict validation in production
	env := os.Getenv("ENV")
	if env == "" {
		env = "development" // Default to development
	}

	skipVerify := env != "production" // Only skip verification if NOT production

	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: skipVerify}

	return &EmailService{
		dialer: dialer,
		sender: sender,
	}
}

// SendOTP sends the 6-digit code to the user
func (s *EmailService) SendOTP(toEmail string, code string) error {
	m := gomail.NewMessage()

	// Set Headers
	// Example: "Mein IDaaS <support@mein-idaas.com>"
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.sender, s.dialer.Username))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Your Verification Code")

	// Set Body (HTML)
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Hello!</h2>
			<p>Your verification code is:</p>
			<h1 style="color: #2d89ef; letter-spacing: 5px;">%s</h1>
			<p>This code will expire in 5 minutes.</p>
			<p>If you did not request this, please ignore this email.</p>
		</div>
	`, code)
	m.SetBody("text/html", body)

	// Send
	if err := s.dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// SendPasswordOTP sends the 6-digit code to the user
func (s *EmailService) SendPasswordOTP(toEmail string, code string) error {
	m := gomail.NewMessage()

	// Set Headers
	// Example: "Mein IDaaS <support@mein-idaas.com>"
	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.sender, s.dialer.Username))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Your Verification Code")

	// Set Body (HTML)
	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Hello!</h2>
			<p>Your password change OTP code is:</p>
			<h1 style="color: #2d89ef; letter-spacing: 5px;">%s</h1>
			<p>This code will expire in 5 minutes.</p>
			<p>If you did not request this, please contact administration team immediately!</p>
		</div>
	`, code)
	m.SetBody("text/html", body)

	// Send
	if err := s.dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// SendForgotPasswordOTP sends the 6-digit OTP code for password reset
func (s *EmailService) SendForgotPasswordOTP(toEmail string, code string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.sender, s.dialer.Username))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Password Reset Code")

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Password Reset Request</h2>
			<p>You requested to reset your password. Use the code below:</p>
			<h1 style="color: #2d89ef; letter-spacing: 5px;">%s</h1>
			<p>This code will expire in 5 minutes.</p>
			<p>If you did not request this, please ignore this email and your password will remain unchanged.</p>
		</div>
	`, code)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}

// SendTemporaryPassword sends the temporary password to the user
func (s *EmailService) SendTemporaryPassword(toEmail string, tempPassword string) error {
	m := gomail.NewMessage()

	m.SetHeader("From", fmt.Sprintf("%s <%s>", s.sender, s.dialer.Username))
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Your Temporary Password")

	body := fmt.Sprintf(`
		<div style="font-family: Arial, sans-serif; padding: 20px;">
			<h2>Password Reset Successful</h2>
			<p>Your password has been successfully reset.</p>
			<p>Your temporary password is:</p>
			<h1 style="color: #2d89ef; letter-spacing: 5px; font-family: monospace;">%s</h1>
			<p style="color: #d32f2f; font-weight: bold;">Please change this password after login for security.</p>
			<p>If you did not request this, please contact support immediately.</p>
		</div>
	`, tempPassword)
	m.SetBody("text/html", body)

	if err := s.dialer.DialAndSend(m); err != nil {
		return err
	}
	return nil
}
