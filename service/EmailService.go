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

	// Fix for common TLS issues (optional but recommended for dev)
	dialer.TLSConfig = &tls.Config{InsecureSkipVerify: true}

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
