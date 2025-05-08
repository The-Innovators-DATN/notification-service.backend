package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"notification-service/internal/config"
	"notification-service/internal/db"
	"notification-service/internal/logging"
	"notification-service/internal/models"
	"notification-service/internal/providers"
)

// WebSocketManager manages WebSocket connections for users
type WebSocketManager struct {
	connections map[int]map[*websocket.Conn]bool // userID -> set of connections
	mutex       sync.Mutex
	logger      *logging.Logger
}

// Service processes alert Tasks and dispatches Notifications
type Service struct {
	db            *db.DB
	logger        *logging.Logger
	config        config.Config
	tasks         chan models.Task
	ctx           context.Context
	cancel        context.CancelFunc
	wg            *sync.WaitGroup
	providerFuncs map[string]func(context.Context, models.Notification, models.ContactPoint) error
	wsManager     *WebSocketManager
}

// New constructs a services Service
func New(db *db.DB, logger *logging.Logger, cfg config.Config) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	svc := &Service{
		db:     db,
		logger: logger,
		config: cfg,
		tasks:  make(chan models.Task, cfg.Notification.QueueSize),
		ctx:    ctx,
		cancel: cancel,
		wsManager: &WebSocketManager{
			connections: make(map[int]map[*websocket.Conn]bool),
			logger:      logger,
		},
	}
	svc.providerFuncs = map[string]func(context.Context, models.Notification, models.ContactPoint) error{
		"email": func(ctx context.Context, notif models.Notification, cp models.ContactPoint) error {
			return providers.SendEmail(ctx, notif, cp, svc.config, logger)
		},
		"telegram": func(ctx context.Context, notif models.Notification, cp models.ContactPoint) error {
			return providers.SendTelegram(ctx, notif, cp, logger, svc.config)
		},
	}
	return svc
}

// Logger exposes the Service's logger
func (s *Service) Logger() *logging.Logger {
	return s.logger
}

// Start launches the worker pool
func (s *Service) Start(wg *sync.WaitGroup) {
	s.wg = wg
	for i := 0; i < s.config.Notification.MaxWorkers; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
}

// QueueTask enqueues a Task for processing
func (s *Service) QueueTask(task models.Task) {
	select {
	case s.tasks <- task:
		s.logger.Infof("Queued task: request_id=%s", task.RequestID)
	default:
		s.logger.Errorf("Queue full, dropping task: request_id=%s", task.RequestID)
	}
}

// worker processes Tasks until context is cancelled
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

// handleTask processes tasks from alert-service and sends notifications
func (s *Service) handleTask(task models.Task) {
	// Parse request ID
	reqID, err := uuid.Parse(task.RequestID)
	if err != nil {
		s.logger.Errorf("Invalid request ID %s: %v", task.RequestID, err)
		return
	}

	// Fetch policies
	policies, err := s.db.GetPoliciesByUserID(s.ctx, task.RecipientID)
	if err != nil {
		s.logger.Errorf("Failed to load policies for user %d: %v", task.RecipientID, err)
		return
	}

	// Process each policy
	for _, pol := range policies {
		if !evaluateCondition(pol.ConditionType, task.Severity, int(pol.Severity)) {
			s.logger.Debugf("Policy %s skipped (severity %d does not satisfy %s %d)", uuid.UUID(pol.ID).String(), task.Severity, pol.ConditionType, pol.Severity)
			continue
		}

		if pol.ContactPoint == nil {
			s.logger.Warnf("Policy %s has no active contact point, skipping", uuid.UUID(pol.ID))
			continue
		}

		// Prepare services body
		body := fmt.Sprintf(
			"%s\nStation: %d\nMetric: %s\nValue: %.2f\nThreshold: %.2f",
			task.Body,
			task.StationID,
			task.MetricName,
			task.Value,
			task.Threshold,
		)

		// Create Notification record
		notif := models.Notification{
			ID:                   reqID,
			CreatedAt:            time.Now(),
			UpdatedAt:            time.Now(),
			Type:                 task.TypeMessage,
			Subject:              task.Subject,
			Body:                 body,
			NotificationPolicyID: pol.ID,
			Status:               "pending",
			RecipientID:          task.RecipientID,
			RequestID:            reqID,
			Silenced:             task.Silenced,
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

		// Persist services
		if err := s.db.CreateNotification(s.ctx, notif); err != nil {
			s.logger.Errorf("CreateNotification failed: %v", err)
			continue
		}

		if notif.Silenced == 0 {
			// Dispatch via provider
			provider := s.providerFuncs[pol.ContactPoint.Type]
			err = provider(s.ctx, notif, *pol.ContactPoint)

			// Send via WebSocket
			message := []byte(fmt.Sprintf("New alert: %s", notif.Subject))
			s.wsManager.SendToUser(task.RecipientID, message)

			// Update status
			final := "success"
			if err != nil {
				final = "failed"
				s.logger.Errorf("Dispatch error via %s: %v", pol.ContactPoint.Type, err)
			}
			_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, final, fmt.Sprintf("%v", err))
			s.logger.Infof("Policy %s dispatched %s via %s", uuid.UUID(pol.ID).String(), final, pol.ContactPoint.Type)
		} else {
			_ = s.db.UpdateNotificationStatus(s.ctx, task.RequestID, "silenced", "Notification silenced, no dispatch")
			s.logger.Infof("Policy %s services silenced", uuid.UUID(pol.ID).String())
		}
	}
}

// evaluateCondition checks if alertSeverity satisfies the policy condition
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

// AddWebSocketConnection adds a WebSocket connection for a user
func (s *Service) AddWebSocketConnection(userID int, conn *websocket.Conn) {
	s.wsManager.AddConnection(userID, conn)
}

// RemoveWebSocketConnection removes a WebSocket connection for a user
func (s *Service) RemoveWebSocketConnection(userID int, conn *websocket.Conn) {
	s.wsManager.RemoveConnection(userID, conn)
}

// AddConnection adds a WebSocket connection
func (m *WebSocketManager) AddConnection(userID int, conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if _, exists := m.connections[userID]; !exists {
		m.connections[userID] = make(map[*websocket.Conn]bool)
	}
	if len(m.connections[userID]) >= 10 { // Giới hạn tối đa 10 kết nối mỗi user
		m.logger.Warnf("Max connections reached for user %d", userID)
		return
	}
	m.connections[userID][conn] = true
	m.logger.Infof("Added WebSocket connection for user %d (total: %d)", userID, len(m.connections[userID]))
}

// RemoveConnection removes a WebSocket connection
func (m *WebSocketManager) RemoveConnection(userID int, conn *websocket.Conn) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if conns, exists := m.connections[userID]; exists {
		delete(conns, conn)
		if len(conns) == 0 {
			delete(m.connections, userID)
		}
		m.logger.Infof("Removed WebSocket connection for user %d (remaining: %d)", userID, len(conns))
	}
}

// SendToUser sends a message to all WebSocket connections of a user
func (m *WebSocketManager) SendToUser(userID int, message []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if conns, exists := m.connections[userID]; exists {
		for conn := range conns {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				m.logger.Errorf("Failed to send WebSocket message to user %d: %v", userID, err)
				delete(conns, conn) // Xóa kết nối lỗi
			}
		}
		if len(conns) == 0 {
			delete(m.connections, userID)
		}
	}
}
