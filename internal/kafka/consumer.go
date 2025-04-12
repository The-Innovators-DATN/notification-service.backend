package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/segmentio/kafka-go"
	"notification-service/internal/logging"
	"notification-service/internal/notification"
	"sync"
	"time"
)

type Config struct {
	Broker string
}

type Consumer struct {
	reader   *kafka.Reader
	svc      *notification.Service
	logger   *logging.Logger
	stopChan chan struct{}
}

func NewConsumer(cfg Config, svc *notification.Service) (*Consumer, error) {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{cfg.Broker},
		Topic:   "alert_notification",
		GroupID: "notification-service",

		MinBytes:    10e3, // 10KB
		MaxBytes:    10e6, // 10MB
		MaxWait:     time.Second * 3,
		StartOffset: kafka.LastOffset,
	})

	return &Consumer{
		reader:   r,
		svc:      svc,
		logger:   svc.Logger(),
		stopChan: make(chan struct{}),
	}, nil
}

func (s *Consumer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.logger.Info("", "Kafka consumer started")

		ctx := context.Background()
		for {
			select {
			case <-s.stopChan:
				s.logger.Info("", "Kafka consumer stopped")
				return
			default:
				msg, err := s.reader.ReadMessage(ctx)
				if err != nil {
					s.logger.Error("", "Read message failed: %v", err)
					continue
				}

				var alert struct {
					AlertID    string `json:"alert_id"`
					AlertName  string `json:"alert_name"`
					Severity   int    `json:"severity"`
					Status     string `json:"status"`
					UserID     int    `json:"user_id"`
					Message    string `json:"message"`
					MetricName string `json:"metric_name"`
					Value      int    `json:"value"`
					Threshold  int    `json:"threshold"`
				}

				if err := json.Unmarshal(msg.Value, &alert); err != nil {
					s.logger.Error("", "Unmarshal message failed: %v", err)
					continue
				}

				if alert.AlertID == "" || alert.Severity < 1 || alert.UserID < 1 {
					s.logger.Error("", "Invalid message: missing alert_id, severity, or user_id")
					continue
				}

				subject := fmt.Sprintf("%s: %s", alert.Status, alert.AlertName)
				body := fmt.Sprintf("Alert: %s\nMessage: %s\nMetric: %s\nValue: %d\nThreshold: %d",
					alert.AlertName, alert.Message, alert.MetricName, alert.Value, alert.Threshold)
				s.svc.QueueNotification("alert_notification", alert.Severity, subject, body, alert.UserID, alert.AlertID)
				s.logger.Info(alert.AlertID, "Processed Kafka message")
			}
		}
	}()
}

func (s *Consumer) Close() {
	close(s.stopChan)
	if err := s.reader.Close(); err != nil {
		s.logger.Error("", "Failed to close Kafka reader: %v", err)
	}
}
