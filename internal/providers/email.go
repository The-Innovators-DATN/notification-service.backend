package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"sync"
	"text/template"
	"time"

	"golang.org/x/time/rate"
	"notification-service/internal/config"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/internal/utils"
)

// emailConfig holds recipient email address parsed from ContactPoint.Configuration.
type emailConfig struct {
	Email string `json:"email"`
}

// emailLimiter is the global rate limiter for email sending
var (
	limiterMu          sync.Mutex
	emailLimiterByUser = map[int]*rate.Limiter{}
)

func getLimiter(uid int, rps int) *rate.Limiter {
	limiterMu.Lock()
	defer limiterMu.Unlock()
	l, ok := emailLimiterByUser[uid]
	if !ok {
		l = rate.NewLimiter(rate.Limit(rps), rps)
		emailLimiterByUser[uid] = l
	}
	return l
}

// SendEmail sends an alert email using SMTP, populating recipient from ContactPoint configuration.
func SendEmail(ctx context.Context, notification models.Notification, cp models.ContactPoint, cfg config.Config, logger *logging.Logger) error {

	// Check rate limit
	if err := getLimiter(notification.RecipientID, cfg.RateLimit.EmailRateLimiter).Wait(ctx); err != nil {
		return fmt.Errorf("email rate limit exceeded: %w", err)
	}

	// Parse recipient email from ContactPoint configuration
	var ec emailConfig
	configBytes, err := json.Marshal(cp.Configuration)
	if err != nil {
		return fmt.Errorf("failed to marshal configuration for user %d: %w", notification.RecipientID, err)
	}
	if err := json.Unmarshal(configBytes, &ec); err != nil {
		return fmt.Errorf("invalid email configuration for user %d: %w", notification.RecipientID, err)
	}
	if ec.Email == "" {
		return fmt.Errorf("email not configured for user %d", notification.RecipientID)
	}

	// Validate SMTP config
	smtpCfg := cfg.Email
	if smtpCfg.SMTPServer == "" || smtpCfg.Username == "" || smtpCfg.Password == "" {
		return fmt.Errorf("incomplete SMTP settings: server/username/password required")
	}
	addr := fmt.Sprintf("%s:%d", smtpCfg.SMTPServer, smtpCfg.SMTPPort)

	tmplPath := filepath.Join("template", "alert_email.html")
	tmplBytes, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read email template file: %w", err)
	}

	tmpl, err := template.New("email").Parse(string(tmplBytes))
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}

	// Prepare template data
	tmplData := struct {
		Subject  string
		FromName string
		Username string
		To       string
		Body     string
		Context  models.AlertContext
		NowYear  int
	}{
		Subject:  notification.Subject,
		FromName: smtpCfg.FromName,
		Username: smtpCfg.Username,
		To:       ec.Email,
		Body:     notification.Body,
		Context:  notification.Context,
		NowYear:  time.Now().Year(),
	}

	var body bytes.Buffer
	if err := tmpl.Execute(&body, tmplData); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	msg := bytes.Buffer{}
	msg.WriteString(fmt.Sprintf("Subject: %s\r\n", notification.Subject))
	msg.WriteString(fmt.Sprintf("From: %s <%s>\r\n", smtpCfg.FromName, smtpCfg.Username))
	msg.WriteString(fmt.Sprintf("To: %s\r\n", ec.Email))
	msg.WriteString("MIME-Version: 1.0\r\n")
	msg.WriteString("Content-Type: text/html; charset=\"UTF-8\"\r\n")
	msg.WriteString("\r\n")
	msg.Write(body.Bytes())

	// Setup authentication
	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.SMTPServer)

	// Retry sending email
	return utils.Retry(logger, 3, time.Second, func() error {
		if err := smtp.SendMail(addr, auth, smtpCfg.Username, []string{ec.Email}, msg.Bytes()); err != nil {
			return fmt.Errorf("error sending email to %s: %w", ec.Email, err)
		}
		return nil
	})
}
