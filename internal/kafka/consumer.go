package kafka

import (
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"notification-service/internal/logging"
	"notification-service/internal/notification"
	"sync"
)

type Config struct {
	Broker string
}

type Consumer struct {
	consumer *kafka.Consumer
	svc      *notification.Service
	logger   *logging.Logger
}

func NewConsumer(cfg Config, svc *notification.Service) (*Consumer, error) {
	c, err := kafka.NewConsumer(&kafka.ConfigMap{
		"bootstrap.servers": cfg.Broker,
		"group.id":          "notification-service",
		"auto.offset.reset": "earliest",
	})
	if err != nil {
		return nil, err
	}
	return &Consumer{consumer: c, svc: svc, logger: svc.Logger()}, nil
}

func (s *Consumer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.logger.Info("", "Kafka consumer started")
		if err := s.consumer.SubscribeTopics([]string{"alert_notification"}, nil); err != nil {
			s.logger.Error("", "Subscribe failed: %v", err)
			return
		}

		for {
			msg, err := s.consumer.ReadMessage(-1)
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
	}()
}

func (s *Consumer) Close() {
	err := s.consumer.Close()
	if err != nil {
		return
	}
}
