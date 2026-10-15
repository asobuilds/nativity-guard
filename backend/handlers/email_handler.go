package handlers

import (
	"crypto/tls"
	"fmt"
	"html"
	"net/mail"
	"net/smtp"
	"os"
	"strings"

	"security-solution/services"
)

type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func getEmailConfig() EmailConfig {
	password := os.Getenv("SMTP_PASS")
	if password == "" {
		password = os.Getenv("SMTP_PASSWORD")
	}

	from := os.Getenv("SMTP_FROM")
	if from == "" {
		from = os.Getenv("SMTP_USER")
	}

	return EmailConfig{
		Host:     strings.TrimSpace(os.Getenv("SMTP_HOST")),
		Port:     strings.TrimSpace(os.Getenv("SMTP_PORT")),
		User:     strings.TrimSpace(os.Getenv("SMTP_USER")),
		Password: password,
		From:     strings.TrimSpace(from),
	}
}

// SendEmail sends an HTML email through the configured SMTP server.
// This implementation expects an SMTP server that supports implicit TLS,
// commonly port 465.
func SendEmail(to, subject, body string) error {
	config := getEmailConfig()

	if config.Host == "" ||
		config.Port == "" ||
		config.User == "" ||
		config.Password == "" ||
		config.From == "" {
		return fmt.Errorf("SMTP configuration missing")
	}

	recipient, err := mail.ParseAddress(to)
	if err != nil {
		return fmt.Errorf("invalid recipient email address")
	}

	sender, err := mail.ParseAddress(config.From)
	if err != nil {
		return fmt.Errorf("invalid SMTP sender address")
	}

	if strings.ContainsAny(subject, "\r\n") {
		return fmt.Errorf("invalid email subject")
	}

	message := strings.Join([]string{
		"From: " + sender.String(),
		"To: " + recipient.String(),
		"Subject: " + subject,
		"MIME-Version: 1.0",
		"Content-Type: text/html; charset=UTF-8",
		"Content-Transfer-Encoding: 8bit",
		"",
		body,
	}, "\r\n")

	conn, err := tls.Dial("tcp", config.Host+":"+config.Port, &tls.Config{
		ServerName: config.Host,
		MinVersion: tls.VersionTLS12,
	})
	if err != nil {
		return fmt.Errorf("SMTP connection failed: %w", err)
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return fmt.Errorf("SMTP client initialization failed: %w", err)
	}
	defer client.Close()

	auth := smtp.PlainAuth(
		"",
		config.User,
		config.Password,
		config.Host,
	)

	if err := client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	if err := client.Mail(sender.Address); err != nil {
		return fmt.Errorf("SMTP sender rejected: %w", err)
	}

	if err := client.Rcpt(recipient.Address); err != nil {
		return fmt.Errorf("SMTP recipient rejected: %w", err)
	}

	writer, err := client.Data()
	if err != nil {
		return fmt.Errorf("SMTP message initialization failed: %w", err)
	}

	if _, err := writer.Write([]byte(message)); err != nil {
		_ = writer.Close()
		return fmt.Errorf("SMTP message transmission failed: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("SMTP message delivery failed: %w", err)
	}

	return nil
}

// SendPasswordResetEmail sends the password-reset link.
// The token is delivered only through the link, not included in the email body
// separately or written to application logs.
func SendPasswordResetEmail(to, resetLink string) error {
	if !strings.HasPrefix(resetLink, "https://") &&
		!strings.HasPrefix(resetLink, "http://localhost") &&
		!strings.HasPrefix(resetLink, "http://127.0.0.1") {
		return fmt.Errorf("invalid password reset URL")
	}

	subject := "Reset your Nativity Guard password"

	body := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>Reset your password</title>
</head>
<body style="margin:0;padding:24px;background:#f4f6f8;font-family:Arial,sans-serif;color:#1f2937">
  <div style="max-width:520px;margin:0 auto;background:#ffffff;padding:32px;border-radius:12px">
    <h1 style="font-size:24px;margin-top:0">Reset your password</h1>
    <p>We received a request to reset the password for your Nativity Guard account.</p>
    <p>
      <a href="%s"
         style="display:inline-block;padding:12px 20px;background:#166534;color:#ffffff;text-decoration:none;border-radius:6px;font-weight:bold">
        Reset password
      </a>
    </p>
    <p>This link expires in <strong>15 minutes</strong> and can only be used once.</p>
    <p>If you did not request this change, you can ignore this email.</p>
    <p style="font-size:12px;color:#6b7280">
      For your security, do not share this link with anyone.
    </p>
  </div>
</body>
</html>`, html.EscapeString(resetLink))

	return SendEmail(to, subject, body)
}

// emailNotifier preserves compatibility with existing application startup code.
type emailNotifier struct{}

// NewEmailNotifier returns the notifier expected by PasswordResetService's
// existing startup configuration.
func NewEmailNotifier() services.ResetNotifier {
	return &emailNotifier{}
}

func (n *emailNotifier) NotifyEmail(toEmail, toName, code string) {
	_ = toName

	body := fmt.Sprintf(
		"<p>Your verification code is: <strong>%s</strong></p>",
		html.EscapeString(code),
	)

	_ = SendEmail(toEmail, "Nativity Guard verification code", body)
}

func (n *emailNotifier) NotifySMS(toPhone, code string) {
	// SMS delivery is not configured in this email handler.
	// Keep the method to preserve compatibility with the existing interface.
	_ = toPhone
	_ = code
}
