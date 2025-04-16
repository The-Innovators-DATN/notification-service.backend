package notification

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/internal/providers"
)

type Service struct {
	db            *db.DB
	logger        *logging.Logger
	config        config.Config
	tasks         chan models.Task
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	providerFuncs map[string]func(task models.Task, cfg config.Config, cp models.ContactPoint) error
}

func New(db *db.DB, logger *logging.Logger, cfg config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:     db,
		logger: logger,
		config: cfg,
		tasks:  make(chan models.Task, cfg.Notification.QueueSize),
		ctx:    ctx,
		cancel: cancel,
		providerFuncs: map[string]func(task models.Task, cfg config.Config, cp models.ContactPoint) error{
			"email":    providers.SendEmail,
			"sms":      providers.SendSMS,
			"telegram": providers.SendTelegram,
		},
	}
}

func (s *Service) Logger() *logging.Logger {
	return s.logger
}

func (s *Service) Start(wg *sync.WaitGroup) {
	s.wg = wg
	for i := 0; i < s.config.Notification.MaxWorkers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
}

func (s *Service) QueueTask(task models.Task) {
	select {
	case s.tasks <- task:
		s.logger.Infof("Task queued: request_id=%s", task.RequestID)
	default:
		s.logger.Errorf("Task queue full, dropping task: request_id=%s", task.RequestID)
	}
}

func (s *Service) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("Worker %d stopped", id)
			return
		case task := <-s.tasks:
			s.processTask(task)
		}
	}
}

func (s *Service) processTask(task models.Task) {
	// Parse UUID từ task.RequestID
	taskRequestID, err := uuid.Parse(task.RequestID)
	if err != nil {
		s.logger.Errorf("Invalid request ID %s: %v", task.RequestID, err)
		return
	}

	// Tạo body bao gồm tất cả thông tin chi tiết
	bodyWithDetails := fmt.Sprintf(
		"%s\nStation ID: %d\nMetric: %s (ID: %d)\nOperator: %s\nThreshold: %.2f (Min: %.2f, Max: %.2f)\nValue: %.2f",
		task.Body, task.StationID, task.MetricName, task.MetricID, task.Operator,
		task.Threshold, task.ThresholdMin, task.ThresholdMax, task.Value,
	)

	// Tạo bản ghi notification với trạng thái pending
	notification := models.Notification{
		ID:                   taskRequestID,
		CreatedAt:            time.Now(),
		Type:                 task.Status,
		Subject:              task.Subject,
		Body:                 bodyWithDetails,
		NotificationPolicyID: [16]byte{},
		Status:               "pending",
		RecipientID:          task.RecipientID,
		RequestID:            taskRequestID,
		LastError:            "",
		LatestStatus:         task.Status,
		StationID:            task.StationID,
		MetricID:             task.MetricID,
		MetricName:           task.MetricName,
		Operator:             task.Operator,
		Threshold:            task.Threshold,
		ThresholdMin:         task.ThresholdMin,
		ThresholdMax:         task.ThresholdMax,
		Value:                task.Value,
	}

	if err := s.db.CreateNotification(s.ctx, notification); err != nil {
		s.logger.Errorf("Failed to create notification for alert_id=%s: %v", task.RequestID, err)
		return
	}

	// Nếu task là "resolved", kiểm tra trạng thái gần nhất
	if task.Status == "resolved" {
		latestNotification, err := s.db.GetLatestNotification(s.ctx, task.RequestID)
		if err == nil {
			// Trường hợp 1: Đã có notification với latest_status = "resolved" và status = "sent"
			if latestNotification.LatestStatus == "resolved" && latestNotification.Status == "sent" {
				// Kiểm tra giá trị (value) để xem có cần gửi lại không
				if latestNotification.Value == task.Value {
					s.logger.Infof("Skipping resolved notification for alert_id=%s, already sent and resolved with same value (%.2f)", task.RequestID, task.Value)
					_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "cancelled", "already sent and resolved with same value")
					return
				}
				s.logger.Infof("Sending resolved notification for alert_id=%s, value changed (old: %.2f, new: %.2f)", task.RequestID, latestNotification.Value, task.Value)
				// Tiếp tục gửi vì giá trị đã thay đổi
			}
			// Trường hợp 2: Đã có notification với latest_status = "resolved" nhưng status = "failed"
			if latestNotification.LatestStatus == "resolved" && latestNotification.Status == "failed" {
				s.logger.Infof("Sending resolved notification for alert_id=%s, previous attempt failed", task.RequestID)
				// Tiếp tục gửi lại
			}
		}
	}

	// Tìm policy
	policy, err := s.db.GetPolicyByTopicAndSeverity(s.ctx, task.Topic, task.Severity)
	if err != nil {
		s.logger.Errorf("Failed to get policy for topic=%s, severity=%d: %v", task.Topic, task.Severity, err)
		_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "failed", err.Error())
		return
	}

	notification.NotificationPolicyID = policy.ID
	_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "pending", "")

	// Lấy contact point
	cp, err := s.db.GetContactPoint(s.ctx, string(policy.ContactPointID[:]))
	if err != nil {
		s.logger.Errorf("Failed to get contact point %s: %v", policy.ContactPointID, err)
		_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "failed", err.Error())
		return
	}

	// Cập nhật task.Body để gửi thông báo
	task.Body = bodyWithDetails

	// Gửi thông báo qua provider
	providerFunc, exists := s.providerFuncs[cp.Type]
	if !exists {
		err := fmt.Errorf("unsupported provider type: %s", cp.Type)
		s.logger.Errorf("Failed to send notification for alert_id=%s: %v", task.RequestID, err)
		_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "failed", err.Error())
		return
	}

	// Gọi providerFunc
	err = providerFunc(task, s.config, cp)
	if err != nil {
		s.logger.Errorf("Failed to send via %s for alert_id=%s: %v", cp.Type, task.RequestID, err)
		_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "failed", err.Error())
		return
	}

	s.logger.Infof("Successfully sent notification via %s for alert_id=%s", cp.Type, task.RequestID)
	_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "sent", "")
}
