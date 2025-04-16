package providers

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"notification-service/internal/config"
	"notification-service/internal/models"
)

type emailConfig struct {
	Email string `json:"email"` // Quay lại một email duy nhất
}

func SendEmail(task models.Task, cfg config.Config, cp models.ContactPoint) error {
	// Parse configuration để lấy email
	var eConfig emailConfig
	if err := json.Unmarshal([]byte(cp.Configuration), &eConfig); err != nil {
		return fmt.Errorf("failed to parse Email configuration for user_id=%d: %w", task.RecipientID, err)
	}

	if eConfig.Email == "" {
		return fmt.Errorf("email not set in configuration for user_id=%d", task.RecipientID)
	}

	// Cấu hình SMTP
	smtpServer := cfg.Email.SMTPServer
	smtpPort := cfg.Email.SMTPPort
	username := cfg.Email.Username
	password := cfg.Email.Password

	if smtpServer == "" || smtpPort == 0 || username == "" || password == "" {
		return fmt.Errorf("missing Email configuration: SMTPServer, SMTPPort, Username, or Password is empty")
	}

	// Tạo nội dung email
	subject := task.Subject
	body := task.Body
	message := fmt.Sprintf("Subject: %s\n\n%s", subject, body)

	// Thiết lập thông tin gửi email
	auth := smtp.PlainAuth("", username, password, smtpServer)
	to := []string{eConfig.Email}
	addr := fmt.Sprintf("%s:%d", smtpServer, smtpPort)

	// Gửi email
	err := smtp.SendMail(addr, auth, username, to, []byte(message))
	if err != nil {
		return fmt.Errorf("failed to send email to %s: %w", eConfig.Email, err)
	}

	return nil
}
