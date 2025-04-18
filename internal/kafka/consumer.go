package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
	"notification-service/internal/logging"
	"notification-service/internal/models"
)

type AlertNotification struct {
	AlertID      string    `json:"alert_id"`
	AlertName    string    `json:"alert_name"`
	StationID    int       `json:"station_id"`
	UserID       int       `json:"user_id"`
	Message      string    `json:"message"`
	Severity     int       `json:"severity"`
	Timestamp    time.Time `json:"timestamp"`
	Status       string    `json:"status"`
	MetricID     int       `json:"metric_id"`
	MetricName   string    `json:"metric_name"`
	Operator     string    `json:"operator"`
	Threshold    float64   `json:"threshold"`
	ThresholdMin float64   `json:"threshold_min"`
	ThresholdMax float64   `json:"threshold_max"`
	Value        float64   `json:"value"`
	PolicyID     string    `json:"policy_id"`
}

type Consumer struct {
	reader           *kafka.Reader
	svc              Service
	logger           *logging.Logger
	latestTimestamps map[string]time.Time
	mu               sync.Mutex
	ctx              context.Context
	cancel           context.CancelFunc
}

type Service interface {
	QueueTask(task models.Task)
	Logger() *logging.Logger
}

func NewConsumer(broker, topic, groupID string, svc Service) (*Consumer, error) {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     []string{broker},
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    1,
		MaxBytes:    10e6,
		StartOffset: kafka.LastOffset,
	})
	ctx, cancel := context.WithCancel(context.Background())
	return &Consumer{
		reader:           reader,
		svc:              svc,
		logger:           svc.Logger(),
		latestTimestamps: make(map[string]time.Time),
		ctx:              ctx,
		cancel:           cancel,
	}, nil
}

func (c *Consumer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.logger.Infof("Kafka consumer started for topic %s with group_id %s", c.reader.Config().Topic, c.reader.Config().GroupID)

		for {
			select {
			case <-c.ctx.Done():
				c.logger.Infof("Consumer stopped due to context cancellation")
				return
			default:
				if err := c.processMessage(c.ctx); err != nil {
					c.logger.Errorf("Failed to process message: %v", err)
					time.Sleep(time.Second) // Tránh spam log quá nhanh
				}
			}
		}
	}()
}

func (c *Consumer) processMessage(ctx context.Context) error {
	const maxRetries = 3
	var lastErr error

	for i := 0; i < maxRetries; i++ {
		select {
		case <-ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
			msg, err := c.reader.ReadMessage(ctx)
			if err == nil {
				var alert AlertNotification
				if err := json.Unmarshal(msg.Value, &alert); err != nil {
					return fmt.Errorf("failed to unmarshal message: %w", err)
				}
				c.logger.Infof("Received alert: %+v", alert)

				c.mu.Lock()
				if lastTimestamp, exists := c.latestTimestamps[alert.AlertID]; exists && !alert.Timestamp.After(lastTimestamp) {
					c.mu.Unlock()
					c.logger.Infof("Skipping outdated message for alert_id %s, timestamp %v", alert.AlertID, alert.Timestamp)
					return nil
				}
				c.latestTimestamps[alert.AlertID] = alert.Timestamp
				c.mu.Unlock()

				task := models.Task{
					RequestID:    alert.AlertID,
					Subject:      alert.AlertName,
					Body:         alert.Message,
					RecipientID:  alert.UserID,
					Severity:     alert.Severity,
					Status:       alert.Status,
					Topic:        c.reader.Config().Topic,
					Timestamp:    alert.Timestamp,
					StationID:    alert.StationID,
					MetricID:     alert.MetricID,
					MetricName:   alert.MetricName,
					Operator:     alert.Operator,
					Threshold:    alert.Threshold,
					ThresholdMin: alert.ThresholdMin,
					ThresholdMax: alert.ThresholdMax,
					Value:        alert.Value,
					PolicyID:     alert.PolicyID,
				}

				c.svc.QueueTask(task)
				c.logger.Infof("Queued task for alert_id %s", task.RequestID)
				return nil
			}

			lastErr = err
			c.logger.Errorf("Failed to read message (attempt %d/%d): %v", i+1, maxRetries, err)
			time.Sleep(time.Second * time.Duration(i+1)) // Backoff
		}
	}

	return fmt.Errorf("failed to read message after %d retries: %w", maxRetries, lastErr)
}

func (c *Consumer) Close() error {
	c.cancel()
	if err := c.reader.Close(); err != nil {
		c.logger.Errorf("Failed to close Kafka reader: %v", err)
		return fmt.Errorf("failed to close Kafka consumer: %w", err)
	}
	c.logger.Infof("Kafka consumer closed")
	return nil
}
