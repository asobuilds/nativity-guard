package handlers

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"net/url"
	"os"
	"strings"

	"security-solution/services"
)

// sendEmailFn is the transport used by email helpers; overridden in tests.
var sendEmailFn = SendEmail

type EmailConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	From     string
}

func getEmailConfig() EmailConfig {
	pass := os.Getenv("SMTP_PASS")
	if pass == "" {
		pass = os.Getenv("SMTP_PASSWORD")
	}
	return EmailConfig{
		Host:     os.Getenv("SMTP_HOST"),
		Port:     os.Getenv("SMTP_PORT"),
		User:     os.Getenv("SMTP_USER"),
		Password: pass,
		From:     os.Getenv("SMTP_FROM"),
	}
}

func SendEmail(to, subject, body string) error {
	config := getEmailConfig()
	if config.Host == "" || config.User == "" || config.Password == "" {
		return fmt.Errorf("SMTP configuration missing")
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/html; charset=UTF-8\r\n\r\n%s",
		config.From, to, subject, body)

	conn, err := tls.Dial("tcp", config.Host+":"+config.Port, &tls.Config{
		ServerName: config.Host,
	})
	if err != nil {
		return err
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, config.Host)
	if err != nil {
		return err
	}
	defer client.Quit()

	auth := smtp.PlainAuth("", config.User, config.Password, config.Host)
	if err = client.Auth(auth); err != nil {
		return err
	}

	if err = client.Mail(config.From); err != nil {
		return err
	}
	if err = client.Rcpt(to); err != nil {
		return err
	}

	writer, err := client.Data()
	if err != nil {
		return err
	}
	_, err = writer.Write([]byte(msg))
	if err != nil {
		return err
	}
	err = writer.Close()
	if err != nil {
		return err
	}

	return nil
}

func SendPasswordResetEmail(to, resetLink string) error {
	subject := "🔐 Reset Your Nativity Guard Password"
	body := fmt.Sprintf(`
		<h1>Password Reset</h1>
		<p>You requested a password reset for your Nativity Guard account.</p>
		<p>Click the link below to reset your password:</p>
		<p><a href="%s">%s</a></p>
		<p>This link expires in 1 hour.</p>
		<br>
		<p>If you didn't request this, please ignore this email.</p>
		<p>Stay safe,</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, resetLink, resetLink)
	return sendEmailFn(to, subject, body)
}

func SendCaseStatusEmail(to, caseTitle, status, caseURL string) error {
	subject := fmt.Sprintf("📋 Case Update: %s", caseTitle)
	body := fmt.Sprintf(`
		<h1>Case Status Updated</h1>
		<p>Your case "<strong>%s</strong>" has been updated.</p>
		<p><strong>New Status:</strong> %s</p>
		<p>View your case: <a href="%s">%s</a></p>
		<br>
		<p>Thank you for using Nativity Guard.</p>
		<p>Stay safe,</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, caseTitle, status, caseURL, caseURL)
	return SendEmail(to, subject, body)
}

func SendSOSConfirmationEmail(to, unitName, message string) error {
	subject := "🚨 SOS Alert Confirmation"
	body := fmt.Sprintf(`
		<h1>🚨 SOS Alert Sent</h1>
		<p>Your SOS alert has been sent successfully.</p>
		<p><strong>Unit notified:</strong> %s</p>
		<p><strong>Message:</strong> %s</p>
		<p>A security unit has been dispatched to your location.</p>
		<br>
		<p>Stay safe,</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, unitName, message)
	return SendEmail(to, subject, body)
}

func SendAnnouncementEmail(to, title, content string) error {
	subject := fmt.Sprintf("📢 New Alert: %s", title)
	body := fmt.Sprintf(`
		<h1>📢 New Announcement</h1>
		<p><strong>%s</strong></p>
		<p>%s</p>
		<br>
		<p>Stay informed, stay safe.</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, title, content)
	return SendEmail(to, subject, body)
}

// SendOTPEmail sends an OTP code via email
func SendOTPEmail(to, code string) error {
	subject := "🔐 Nativity Guard - OTP Verification"
	body := fmt.Sprintf(`
		<h1>OTP Verification</h1>
		<p>Your OTP code for Nativity Guard is:</p>
		<h2 style="font-size: 32px; letter-spacing: 4px; background: #f0f0f0; padding: 12px; text-align: center;">%s</h2>
		<p>This code expires in 10 minutes.</p>
		<br>
		<p>If you didn't request this, please ignore this email.</p>
		<p>Stay safe,</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, code)
	return SendEmail(to, subject, body)
}

// emailNotifier implements services.ResetNotifier using the SMTP-backed
// SendEmail helper above. Registered once at startup from main.go.
type emailNotifier struct{}

// NotifyEmail delivers the reset code via SMTP. If toEmail is empty, this
// is a no-op — we never send to "".
func (n *emailNotifier) NotifyEmail(toEmail, toName, code string) {
	if toEmail == "" {
		return
	}
	_ = SendPasswordResetTokenEmail(toEmail, code)
}

// NotifySMS delivers the reset code via the existing SMS service. If
// toPhone is empty, this is a no-op.
func (n *emailNotifier) NotifySMS(toPhone, code string) {
	if toPhone == "" {
		return
	}
	_ = services.SendSMS(toPhone, "Your Nativity Guard reset code: "+code+" (valid 1 hour)")
}

// NewEmailNotifier returns a services.ResetNotifier backed by SMTP/SMS.
func NewEmailNotifier() *emailNotifier {
	return &emailNotifier{}
}

// SendPasswordResetTokenEmail delivers the raw reset token via email.
// When BASE_URL is configured, it builds a clickable reset link and reuses
// SendPasswordResetEmail. When BASE_URL is unset, it falls back to showing
// the raw token inline.
func SendPasswordResetTokenEmail(to, token string) error {
	if baseURL := strings.TrimSpace(os.Getenv("BASE_URL")); baseURL != "" {
		resetLink := fmt.Sprintf("%s/auth/reset-password?token=%s", strings.TrimRight(baseURL, "/"), url.QueryEscape(token))
		return SendPasswordResetEmail(to, resetLink)
	}

	subject := "🔐 Reset Your Nativity Guard Password"
	body := fmt.Sprintf(`
		<h1>Password Reset</h1>
		<p>You requested a password reset for your Nativity Guard account.</p>
		<p>Your reset token is:</p>
		<h2 style="font-size: 28px; letter-spacing: 2px; background: #f0f0f0; padding: 12px; text-align: center; font-family: monospace;">%s</h2>
		<p>This token expires in 1 hour.</p>
		<br>
		<p>If you didn't request this, please ignore this email.</p>
		<p>Stay safe,</p>
		<p><strong>Nativity Guard Team</strong></p>
	`, token)
	return sendEmailFn(to, subject, body)
}
