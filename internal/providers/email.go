package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/smtp"
	"text/template"

	"notification-service/internal/config"
	"notification-service/internal/models"
)

// emailConfig holds recipient email address parsed from ContactPoint.Configuration.
type emailConfig struct {
	Email string `json:"email"`
}

// emailTemplate defines the structure of the email body with optional context fields.
const emailTemplate = `Subject: {{ .Subject }}
` +
	`From: {{ .FromName }} <{{ .Username }}>
` +
	`To: {{ .To }}
` +
	`MIME-Version: 1.0
` +
	`Content-Type: text/plain; charset="utf-8"
` +
	`
` +
	`Alert Details:
` +
	`- Station ID: {{ .Context.StationID }}
` +
	`- Metric: {{ .Context.MetricName }} (ID {{ .Context.MetricID }})
` +
	`- Operator: {{ .Context.Operator }}
` +
	`- Threshold: {{ .Context.ThresholdMin }} - {{ .Context.ThresholdMax }} (target {{ .Context.Threshold }})
` +
	`- Value: {{ .Context.Value }}
` +
	`
` +
	`Message:
` +
	`{{ .Body }}`

// SendEmail sends an alert email using SMTP, populating recipient from ContactPoint configuration.
func SendEmail(ctx context.Context, notification models.Notification, cp models.ContactPoint, cfg config.Config) error {
	// Parse recipient email from ContactPoint configuration
	var ec emailConfig
	if err := json.Unmarshal([]byte(cp.Configuration), &ec); err != nil {
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

	// Prepare template data
	tmplData := struct {
		Subject  string
		FromName string
		Username string
		To       string
		Body     string
		Context  models.AlertContext
	}{
		Subject:  notification.Subject,
		FromName: smtpCfg.FromName,
		Username: smtpCfg.Username,
		To:       ec.Email,
		Body:     notification.Body,
		Context:  notification.Context,
	}

	// Render email
	tmpl, err := template.New("email").Parse(emailTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse email template: %w", err)
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplData); err != nil {
		return fmt.Errorf("failed to execute email template: %w", err)
	}

	// Setup authentication
	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.SMTPServer)

	// Send email
	if err := smtp.SendMail(addr, auth, smtpCfg.Username, []string{ec.Email}, buf.Bytes()); err != nil {
		return fmt.Errorf("error sending email to %s: %w", ec.Email, err)
	}
	return nil
}
