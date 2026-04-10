package email

import (
	"fmt"
	"net/smtp"
)

// Sender is the interface for sending emails.
type Sender interface {
	SendVerification(to, verificationURL string) error
}

// SMTPSender sends emails via SMTP.
type SMTPSender struct {
	host     string
	port     string
	username string
	password string
	from     string
}

func NewSMTPSender(host, port, username, password, from string) *SMTPSender {
	return &SMTPSender{
		host:     host,
		port:     port,
		username: username,
		password: password,
		from:     from,
	}
}

func (s *SMTPSender) SendVerification(to, verificationURL string) error {
	subject := "Verify your Superset account"
	body := fmt.Sprintf(
		"Welcome! Please verify your account by clicking the link below:\n\n%s\n\nThis link expires in 24 hours.",
		verificationURL,
	)
	msg := []byte(fmt.Sprintf(
		"From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s",
		s.from, to, subject, body,
	))

	addr := fmt.Sprintf("%s:%s", s.host, s.port)
	var auth smtp.Auth
	if s.username != "" {
		auth = smtp.PlainAuth("", s.username, s.password, s.host)
	}
	return smtp.SendMail(addr, auth, s.from, []string{to}, msg)
}

// NoOpSender discards all emails — useful for testing and local development.
type NoOpSender struct{}

func (NoOpSender) SendVerification(_, _ string) error { return nil }
