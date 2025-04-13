package notification

import (
	"context"
	"errors"
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

const (
	MaxRetries      = 3
	MaxWorkers      = 10
	MaxRetryWorkers = 2
	QueueSize       = 500
)

var (
	ErrNoRecipients      = errors.New("no recipients configured")
	ErrProviderNotFound  = errors.New("unknown provider")
	ErrPermanentFailure  = errors.New("permanent error")
	ErrPolicyNotFound    = errors.New("policy not found")
	ErrInvalidParameters = errors.New("invalid parameters")
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
	wg         *sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func New(db *db.DB, logger *logging.Logger, cfg config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:         db,
		logger:     logger,
		config:     cfg,
		tasks:      make(chan Task, QueueSize),
		retryTasks: make(chan Task, QueueSize),
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (s *Service) Logger() *logging.Logger {
	return s.logger
}

func (s *Service) Start(wg *sync.WaitGroup) {
	s.wg = wg
	for i := 0; i < MaxWorkers; i++ {
		s.wg.Add(1)
		go s.startWorker(i)
	}

	for i := 0; i < MaxRetryWorkers; i++ {
		s.wg.Add(1)
		go s.startRetryWorker(i)
	}
}

func (s *Service) Stop() {
	s.cancel()
	close(s.tasks)
	close(s.retryTasks)
}

func (s *Service) QueueNotification(topic string, severity int, subject, body string, recipientID int, requestID string) error {
	if topic == "" || severity < 0 || subject == "" || body == "" || recipientID <= 0 || requestID == "" {
		s.logger.Error(requestID, "Invalid parameters: topic=%s, severity=%d, subject=%s, body=%s, recipientID=%d, requestID=%s",
			topic, severity, subject, body, recipientID, requestID)
		return ErrInvalidParameters
	}

	select {
	case s.tasks <- Task{
		Topic:       topic,
		Severity:    severity,
		Subject:     subject,
		Body:        body,
		RecipientID: recipientID,
		RequestID:   requestID,
		RetryCount:  0,
	}:
		return nil

	default:
		s.logger.Error(requestID, "Queue task channel is full")
		return errors.New("queue task channel is full")
	}
}

func (s *Service) startWorker(workerID int) {
	defer s.wg.Done()
	s.logger.Info("", "Worker %d started", workerID)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("", "Worker %d stopped", workerID)
			return
		case task, ok := <-s.tasks:
			if !ok {
				s.logger.Info("", "Worker %d stopped", workerID)
				return
			}
			ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
			s.handleTask(ctx, task)
			cancel()
		}
	}
}

func (s *Service) startRetryWorker(workerID int) {
	defer s.wg.Done()
	s.logger.Info("", "Retry worker %d started", workerID)

	for {
		select {
		case <-s.ctx.Done():
			s.logger.Info("", "Retry worker %d stopped", workerID)
			return
		case task, ok := <-s.retryTasks:
			if !ok {
				s.logger.Info("", "Retry worker %d stopped", workerID)
				return
			}
			ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
			s.handleRetryTask(ctx, task)
			cancel()
		}
	}
}

func (s *Service) handleTask(ctx context.Context, task Task) {
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
	}
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
