package notification

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"math"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/pkg/email"
	"notification-service/pkg/sms"
	"notification-service/pkg/telegram"
	"strings"
	"sync"
	"time"
)

type Task struct {
	Topic       string
	Severity    int
	Subject     string
	Body        string
	RecipientID int
	RequestID   string
	RetryCount  int
}

type Service struct {
	db         *db.DB
	logger     *logging.Logger
	config     config.Config
	tasks      chan Task
	retryTasks chan Task
}

const (
	MaxRetries = 3
)

func New(db *db.DB, logger *logging.Logger, cfg config.Config) *Service {
	return &Service{
		db:         db,
		logger:     logger,
		config:     cfg,
		tasks:      make(chan Task, 500),
		retryTasks: make(chan Task, 500),
	}
}

func (s *Service) Logger() *logging.Logger {
	return s.logger
}

func (s *Service) Start(wg *sync.WaitGroup) {
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			s.logger.Info("", "Worker %d started", workerID)
			for task := range s.tasks {
				s.handleTask(task)
			}
		}(i)
	}

	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(retryID int) {
			defer wg.Done()
			s.logger.Info("", "Retry Manager %d started", retryID)
			for task := range s.retryTasks {
				s.handleRetryTask(task)
			}
		}(i)
	}
}

func (s *Service) QueueNotification(topic string, severity int, subject, body string, recipientID int, requestID string) {
	s.tasks <- Task{
		Topic:       topic,
		Severity:    severity,
		Subject:     subject,
		Body:        body,
		RecipientID: recipientID,
		RequestID:   requestID,
		RetryCount:  0,
	}
}

func (s *Service) handleTask(task Task) {
	n := models.Notification{
		ID:           uuid.New().String(),
		CreatedAt:    time.Now(),
		Type:         "alert",
		Subject:      task.Subject,
		Body:         task.Body,
		Status:       "pending",
		RecipientID:  task.RecipientID,
		RequestID:    task.RequestID,
		RetryCount:   0,
		LatestStatus: task.Subject[:6],
	}
	if err := s.db.CreateNotification(context.Background(), n); err != nil {
		s.logger.Error(task.RequestID, "Create notification failed: %v", err)
		return
	}

	s.logger.Debug(task.RequestID, "Fetching policy for severity %d", task.Severity)
	policy, contact, err := s.db.GetPolicyAndContact(context.Background(), task.Topic, task.Severity)
	if err != nil {
		s.logger.Error(task.RequestID, "Query policy failed: %v", err)
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "failed", err.Error()); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		return
	}
	n.NotificationPolicyID = policy.ID

	err = s.send(contact.Type, contact.Configuration, task.Subject, task.Body)
	if err == nil {
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "sent", ""); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		s.logger.Info(task.RequestID, "Sent via %s", contact.Type)
	} else {
		s.logger.Warn(task.RequestID, "Retry queued (1/%d): %v", MaxRetries, err)
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "retrying", err.Error()); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		task.RetryCount = 1
		s.retryTasks <- task
	}
}

func (s *Service) handleRetryTask(task Task) {
	// Exponential backoff: 30s, 60s, 120s
	interval := time.Duration(math.Pow(2, float64(task.RetryCount-1))) * 30 * time.Second
	<-time.After(interval)

	s.logger.Debug(task.RequestID, "Checking latest status before retry")
	latestStatus, err := s.db.GetLatestStatus(context.Background(), task.RequestID)
	if err != nil {
		s.logger.Error(task.RequestID, "Get latest status failed: %v", err)
		return
	}
	if latestStatus == "resolved" && task.Subject[:6] == "Alert:" {
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "cancelled", "Resolved received"); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		s.logger.Info(task.RequestID, "Retry cancelled due to resolved")
		return
	}

	_, contact, err := s.db.GetPolicyAndContact(context.Background(), task.Topic, task.Severity)
	if err != nil {
		s.logger.Error(task.RequestID, "Query policy failed: %v", err)
		return
	}

	err = s.send(contact.Type, contact.Configuration, task.Subject, task.Body)
	if err == nil {
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "sent", ""); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		s.logger.Info(task.RequestID, "Sent via %s", contact.Type)
	} else if task.RetryCount < MaxRetries {
		// Stop retry early for permanent errors
		if isPermanentError(err) {
			s.logger.Error(task.RequestID, "Permanent error, stopping retry: %v", err)
			if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "failed", err.Error()); err != nil {
				s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
			}
			return
		}
		s.logger.Warn(task.RequestID, "Retry queued (%d/%d): %v", task.RetryCount+1, MaxRetries, err)
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "retrying", err.Error()); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
		task.RetryCount++
		s.retryTasks <- task
	} else {
		// Try fallback provider
		fallback, ok := contact.Configuration["fallback"].(string)
		if ok && fallback != "" {
			s.logger.Info(task.RequestID, "Max retries reached, trying fallback: %s", fallback)
			fallbackContact, err := s.getFallbackContact(fallback)
			if err == nil {
				err = s.send(fallbackContact.Type, fallbackContact.Configuration, task.Subject, task.Body)
				if err == nil {
					if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "sent", ""); err != nil {
						s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
					}
					s.logger.Info(task.RequestID, "Sent via fallback %s", fallbackContact.Type)
					return
				}
			}
		}
		s.logger.Error(task.RequestID, "Failed after %d retries: %v", MaxRetries, err)
		if err := s.db.UpdateNotificationStatus(context.Background(), task.RequestID, "failed", err.Error()); err != nil {
			s.logger.Error(task.RequestID, "Update notification status failed: %v", err)
		}
	}
}

func (s *Service) getFallbackContact(providerType string) (models.ContactPoint, error) {
	rows, err := s.db.Conn.Query(context.Background(), `
		SELECT id, name, user_id, type, configuration, status, created_at
		FROM contact_points WHERE type = $1 AND status = 'active' LIMIT 1`, providerType)
	if err != nil {
		return models.ContactPoint{}, err
	}

	var cp models.ContactPoint
	for rows.Next() {
		err := rows.Scan(&cp.ID, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt)
		if err != nil {
			rows.Close()
			return models.ContactPoint{}, err
		}
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return models.ContactPoint{}, err
	}
	rows.Close()

	if cp.ID == "" {
		return models.ContactPoint{}, fmt.Errorf("no active contact point for %s", providerType)
	}
	return cp, nil
}

func isPermanentError(err error) bool {
	errStr := err.Error()
	return strings.Contains(errStr, "Invalid credentials") ||
		strings.Contains(errStr, "Invalid to number") ||
		strings.Contains(errStr, "Chat ID not found")
}

func (s *Service) send(providerType string, config map[string]interface{}, subject, body string) error {
	switch providerType {
	case "email":
		recipientsRaw, ok := config["recipients"].([]interface{})
		if !ok || len(recipientsRaw) == 0 {
			return fmt.Errorf("no recipients configured")
		}
		recipients := make([]string, len(recipientsRaw))
		for i, v := range recipientsRaw {
			recipients[i] = v.(string)
		}
		for _, recipient := range recipients {
			err := email.Send(
				s.config.Email.SMTPServer,
				s.config.Email.SMTPPort,
				s.config.Email.Username,
				s.config.Email.Password,
				recipient,
				subject,
				body,
			)
			if err != nil {
				s.logger.Error("", "Failed to send email to %s: %v", recipient, err)
				return err
			}
		}
		return nil
	case "telegram":
		chatIDsRaw, ok := config["chat_ids"].([]interface{})
		if !ok || len(chatIDsRaw) == 0 {
			return fmt.Errorf("no chat_ids configured")
		}
		chatIDs := make([]int64, len(chatIDsRaw))
		for i, v := range chatIDsRaw {
			chatIDs[i] = int64(v.(float64))
		}
		return telegram.Send(s.config.Telegram.BotToken, chatIDs, body)
	case "sms":
		toNumbersRaw, ok := config["to_numbers"].([]interface{})
		if !ok || len(toNumbersRaw) == 0 {
			return fmt.Errorf("no to_numbers configured")
		}
		toNumbers := make([]string, len(toNumbersRaw))
		for i, v := range toNumbersRaw {
			toNumbers[i] = v.(string)
		}
		for _, toNumber := range toNumbers {
			err := sms.Send(
				s.config.SMS.AccountSID,
				s.config.SMS.AuthToken,
				s.config.SMS.FromNumber,
				toNumber,
				body,
			)
			if err != nil {
				s.logger.Error("", "Failed to send SMS to %s: %v", toNumber, err)
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("unknown provider: %s", providerType)
	}
}
