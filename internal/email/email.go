package email

import (
	"log"
)

// EmailService handles sending emails
type EmailService struct {
	fromEmail string
	fromName  string
	enabled   bool
}

// NewEmailService creates a new email service
func NewEmailService(fromEmail, fromName string, enabled bool) *EmailService {
	return &EmailService{
		fromEmail: fromEmail,
		fromName:  fromName,
		enabled:   enabled,
	}
}

// Config for an email message
type EmailConfig struct {
	To       string
	Subject  string
	Template string
	Data     map[string]interface{}
}

// SendEmail sends an email
func (s *EmailService) SendEmail(config EmailConfig) error {
	if !s.enabled {
		log.Printf("[MOCK EMAIL] To: %s, Subject: %s, Template: %s",
			config.To, config.Subject, config.Template)
		log.Printf("[MOCK EMAIL] Data: %v", config.Data)
		return nil
	}

	// In production, connect to a real email service like Sendgrid, Mailgun, etc.
	// For now, we'll just log the email
	log.Printf("Would send email to %s with subject '%s'", config.To, config.Subject)
	return nil
}

// SendPasswordResetEmail sends a password reset email
func (s *EmailService) SendPasswordResetEmail(email, resetLink string) error {
	return s.SendEmail(EmailConfig{
		To:       email,
		Subject:  "Reset Your Password",
		Template: "password_reset",
		Data: map[string]interface{}{
			"ResetLink": resetLink,
		},
	})
}

// SendWelcomeEmail sends a welcome email to new users
func (s *EmailService) SendWelcomeEmail(email, name string) error {
	return s.SendEmail(EmailConfig{
		To:       email,
		Subject:  "Welcome to Tickit",
		Template: "welcome",
		Data: map[string]interface{}{
			"Name": name,
		},
	})
}

// SendAccountVerificationEmail sends an email for account verification
func (s *EmailService) SendAccountVerificationEmail(email, verificationLink string) error {
	return s.SendEmail(EmailConfig{
		To:       email,
		Subject:  "Verify Your Account",
		Template: "account_verification",
		Data: map[string]interface{}{
			"VerificationLink": verificationLink,
		},
	})
}
