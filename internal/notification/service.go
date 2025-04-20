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

// Service processes alert Tasks and dispatches Notifications according to user policies.
type Service struct {
	db            *db.DB
	logger        *logging.Logger
	config        config.Config
	tasks         chan models.Task
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	providerFuncs map[string]func(context.Context, models.Notification, models.ContactPoint) error
}

// New constructs a notification Service .
func New(db *db.DB, logger *logging.Logger, cfg config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		db:     db,
		logger: logger,
		config: cfg,
		tasks:  make(chan models.Task, cfg.Notification.QueueSize),
		ctx:    ctx,
		cancel: cancel,
	}
	// initialize provider functions, injecting config where needed
	svc.providerFuncs = map[string]func(context.Context, models.Notification, models.ContactPoint) error{
		"email": func(ctx context.Context, notif models.Notification, cp models.ContactPoint) error {
			return providers.SendEmail(ctx, notif, cp, svc.config)
		},
		"telegram": func(ctx context.Context, notif models.Notification, cp models.ContactPoint) error {
			return providers.SendTelegram(ctx, notif, cp)
		},
	}
	return svc
}

// Logger exposes the Service's logger to the Kafka consumer or caller.
func (s *Service) Logger() *logging.Logger {
	return s.logger
}

// Start launches the worker pool.
func (s *Service) Start(wg *sync.WaitGroup) {
	s.wg = wg
	for i := 0; i < s.config.Notification.MaxWorkers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
}

// QueueTask enqueues a Task for processing.
func (s *Service) QueueTask(task models.Task) {
	select {
	case s.tasks <- task:
		s.logger.Infof("Queued task: request_id=%s", task.RequestID)
	default:
		s.logger.Errorf("Queue full, dropping task: request_id=%s", task.RequestID)
	}
}

// worker processes Tasks until context is cancelled.
func (s *Service) worker(id int) {
	defer s.wg.Done()
	for {
		select {
		case <-s.ctx.Done():
			s.logger.Infof("Worker %d stopped", id)
			return
		case task := <-s.tasks:
			s.handleTask(task)
		}
	}
}

// handleTask retrieves policies, evaluates conditions, creates Notifications, and dispatches.
func (s *Service) handleTask(task models.Task) {
	// parse the request ID
	reqID, err := uuid.Parse(task.RequestID)
	if err != nil {
		s.logger.Errorf("Invalid request ID %s: %v", task.RequestID, err)
		return
	}

	// fetch all active policies for the user
	policies, err := s.db.GetPoliciesByUserID(s.ctx, task.RecipientID)
	if err != nil {
		s.logger.Errorf("Failed to load policies for user %d: %v", task.RecipientID, err)
		return
	}

	// evaluate each policy
	for _, pol := range policies {
		// check severity against policy condition
		if !evaluateCondition(pol.ConditionType, task.Severity, int(pol.Severity)) {
			s.logger.Debugf("Policy %s skipped (severity %d does not satisfy %s %d)", uuid.UUID(pol.ID).String(), task.Severity, pol.ConditionType, pol.Severity)
			continue
		}

		if pol.ContactPoint == nil {
			s.logger.Warnf("Policy %s has no active contact point, skipping", uuid.UUID(pol.ID))
			continue
		}

		// prepare notification body with context
		body := fmt.Sprintf(
			"%s\nStation: %d\nMetric: %s\nValue: %.2f\nThreshold: %.2f",
			task.Body,
			task.StationID,
			task.MetricName,
			task.Value,
			task.Threshold,
		)

		// create Notification record
		notif := models.Notification{
			ID:                   reqID,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			Type:                 task.TypeMessage, // "alert" or "resolved"
			Subject:              task.Subject,
			Body:                 body,
			NotificationPolicyID: pol.ID,
			Status:               "pending", // initial status
			RecipientID:          task.RecipientID,
			RequestID:            reqID,
			Context: models.AlertContext{
				StationID:    task.StationID,
				MetricID:     task.MetricID,
				MetricName:   task.MetricName,
				Operator:     task.Operator,
				Threshold:    task.Threshold,
				ThresholdMin: task.ThresholdMin,
				ThresholdMax: task.ThresholdMax,
				Value:        task.Value,
			},
		}

		// persist notification
		if err := s.db.CreateNotification(s.ctx, notif); err != nil {
			s.logger.Errorf("CreateNotification failed: %v", err)
			continue
		}

		// dispatch via provider
		provider := s.providerFuncs[pol.ContactPoint.Type]
		err = provider(s.ctx, notif, *pol.ContactPoint)

		// finalize status
		final := "success"
		if err != nil {
			final = "failed"
			s.logger.Errorf("Dispatch error via %s: %v", pol.ContactPoint.Type, err)
		}
		_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, final, fmt.Sprintf("%v", err))

		s.logger.Infof("Policy %s dispatched %s via %s", uuid.UUID(pol.ID).String(), final, pol.ContactPoint.Type)
	}
}

// evaluateCondition checks if alertSeverity satisfies the policy condition.
func evaluateCondition(cond string, alertSeverity, policySeverity int) bool {
	switch cond {
	case "EQ":
		return alertSeverity == policySeverity
	case "NEQ":
		return alertSeverity != policySeverity
	case "GT":
		return alertSeverity > policySeverity
	case "GTE":
		return alertSeverity >= policySeverity
	case "LT":
		return alertSeverity < policySeverity
	case "LTE":
		return alertSeverity <= policySeverity
	default:
		return false
	}
}
