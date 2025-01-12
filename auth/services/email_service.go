package services

import (
	"fmt"
	"net/smtp"
)

type EmailService interface {
	SendPasswordResetEmail(to, resetToken string) error
	SendVerificationEmail(to, verificationToken string) error
}

type emailServiceImpl struct {
	config SMTPConfig
}

type SMTPConfig struct {
	Host     string
	Port     int
	Username string
	Password string
	From     string
}

func NewEmailService(config SMTPConfig) EmailService {
	return &emailServiceImpl{
		config: config,
	}
}

func (s *emailServiceImpl) SendPasswordResetEmail(to, resetToken string) error {
	subject := "Password Reset Request"
	body := fmt.Sprintf(`
Hello,

You have requested to reset your password. Please use the following token:

%s

If you did not request this, please ignore this email.

Best regards,
Your App Team
`, resetToken)

	return s.sendEmail(to, subject, body)
}

func (s *emailServiceImpl) SendVerificationEmail(to, verificationToken string) error {
	subject := "Verify Your Email"
	body := fmt.Sprintf(`
Hello,

Please verify your email address by using the following token:

%s

Best regards,
Your App Team
`, verificationToken)

	return s.sendEmail(to, subject, body)
}

func (s *emailServiceImpl) sendEmail(to, subject, body string) error {
	addr := fmt.Sprintf("%s:%d", s.config.Host, s.config.Port)
	msg := fmt.Sprintf("From: %s\r\n"+
		"To: %s\r\n"+
		"Subject: %s\r\n"+
		"\r\n"+
		"%s", s.config.From, to, subject, body)

	var auth smtp.Auth
	if s.config.Username != "" {
		auth = smtp.PlainAuth("", s.config.Username, s.config.Password, s.config.Host)
	}

	return smtp.SendMail(addr, auth, s.config.From, []string{to}, []byte(msg))
}
