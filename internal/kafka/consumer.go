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

// AlertNotification represents the payload consumed from Kafka.
type AlertNotification struct {
	AlertID      string    `json:"alert_id"`
	AlertName    string    `json:"alert_name"`
	StationID    int       `json:"station_id"`
	UserID       int64     `json:"user_id"`
	Message      string    `json:"message"`
	Severity     int       `json:"severity"`
	Timestamp    time.Time `json:"timestamp"`
	TypeMessage  string    `json:"type_message"`
	MetricID     int       `json:"metric_id"`
	MetricName   string    `json:"metric_name"`
	Operator     string    `json:"operator"`
	Threshold    float64   `json:"threshold"`
	ThresholdMin float64   `json:"threshold_min"`
	ThresholdMax float64   `json:"threshold_max"`
	Value        float64   `json:"value"`
}

// Consumer reads AlertNotification messages and enqueues tasks.
type Consumer struct {
	reader   *kafka.Reader
	svc      Service
	logger   *logging.Logger
	mu       sync.Mutex
	lastSeen map[string]time.Time
	ctx      context.Context
	cancel   context.CancelFunc
}

// Service defines dependencies needed by Consumer.
type Service interface {
	QueueTask(models.Task)
	Logger() *logging.Logger
}

// NewConsumer constructs a new Consumer.
func NewConsumer(brokers []string, topic, groupID string, svc Service) *Consumer {
	ctx, cancel := context.WithCancel(context.Background())
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:        brokers,
		Topic:          topic,
		GroupID:        groupID,
		MinBytes:       1,
		MaxBytes:       10e6,
		StartOffset:    kafka.LastOffset,
		CommitInterval: time.Second,
	})
	return &Consumer{
		reader:   reader,
		svc:      svc,
		logger:   svc.Logger(),
		lastSeen: make(map[string]time.Time),
		ctx:      ctx,
		cancel:   cancel,
	}
}

// Start launches the consume loop in a goroutine.
func (c *Consumer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.logger.Infof("Kafka consumer started for topic %s", c.reader.Config().Topic)
		for {
			if err := c.consumeNext(); err != nil {
				if c.ctx.Err() != nil {
					c.logger.Infof("Consumer context canceled, exiting")
					return
				}
				c.logger.Errorf("consume error: %v", err)
				time.Sleep(time.Second)
			}
		}
	}()
}

// Close stops consumption and closes the reader.
func (c *Consumer) Close() error {
	c.cancel()
	if err := c.reader.Close(); err != nil {
		c.logger.Errorf("error closing kafka reader: %v", err)
		return fmt.Errorf("close error: %w", err)
	}
	c.logger.Infof("Kafka consumer closed")
	return nil
}

// consumeNext reads one message with retry and processes it.
func (c *Consumer) consumeNext() error {
	msg, err := c.reader.FetchMessage(c.ctx)
	if err != nil {
		return fmt.Errorf("fetch message: %w", err)
	}
	defer func() {
		// commit within defer ensures commit even on panic
		if err := c.reader.CommitMessages(c.ctx, msg); err != nil {
			c.logger.Errorf("commit failed: %v", err)
		}
	}()

	var alert AlertNotification
	if err := json.Unmarshal(msg.Value, &alert); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	c.logger.Infof("Received alert %s at %s", alert.AlertID, alert.Timestamp)

	// Deduplicate by timestamp
	c.mu.Lock()
	last, ok := c.lastSeen[alert.AlertID]
	if ok && !alert.Timestamp.After(last) {
		c.mu.Unlock()
		c.logger.Infof("outdated alert %s (seen %s)", alert.AlertID, last)
		return nil
	}
	c.lastSeen[alert.AlertID] = alert.Timestamp
	c.mu.Unlock()

	// Enqueue task
	task := models.Task{
		RequestID:    alert.AlertID,
		Subject:      alert.AlertName,
		Body:         alert.Message,
		RecipientID:  alert.UserID,
		Severity:     alert.Severity,
		TypeMessage:  alert.TypeMessage,
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
	}
	c.svc.QueueTask(task)
	c.logger.Infof("Task queued for alert %s", alert.AlertID)

	return nil
}
