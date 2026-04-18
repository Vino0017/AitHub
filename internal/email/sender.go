package email

import (
	"fmt"
	"net/smtp"
	"os"
)

// Sender handles sending verification emails.
type Sender struct {
	host     string
	port     string
	from     string
	password string
	enabled  bool
}

// NewSender creates an email sender from environment variables.
// Required env: SMTP_HOST, SMTP_PORT, SMTP_FROM, SMTP_PASSWORD
func NewSender() *Sender {
	host := os.Getenv("SMTP_HOST")
	port := os.Getenv("SMTP_PORT")
	from := os.Getenv("SMTP_FROM")
	pass := os.Getenv("SMTP_PASSWORD")

	if port == "" {
		port = "587"
	}

	return &Sender{
		host:     host,
		port:     port,
		from:     from,
		password: pass,
		enabled:  host != "" && from != "",
	}
}

// IsEnabled returns whether SMTP is configured.
func (s *Sender) IsEnabled() bool {
	return s.enabled
}

// SendVerificationCode sends a 6-char verification code to the given email.
func (s *Sender) SendVerificationCode(to, code, namespace string) error {
	if !s.enabled {
		return fmt.Errorf("SMTP not configured")
	}

	subject := "SkillHub Verification Code"
	body := fmt.Sprintf(`Hi,

Your SkillHub verification code for namespace "%s" is:

    %s

This code expires in 10 minutes.

— SkillHub Registry`, namespace, code)

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s",
		s.from, to, subject, body)

	auth := smtp.PlainAuth("", s.from, s.password, s.host)
	addr := s.host + ":" + s.port

	return smtp.SendMail(addr, auth, s.from, []string{to}, []byte(msg))
}
