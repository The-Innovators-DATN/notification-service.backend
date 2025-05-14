package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/IBM/sarama"
	"notification-service/internal/logging"
	"notification-service/internal/models"
)

// AlertNotification represents the payload consumed from Kafka.
type AlertNotification struct {
	AlertID      string    `json:"alert_id"`
	AlertName    string    `json:"alert_name"`
	StationID    int       `json:"station_id"`
	UserID       int       `json:"user_id"`
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

// Custom unmarshalling for AlertNotification to handle timestamp as an array.
func (a *AlertNotification) UnmarshalJSON(data []byte) error {
	type Alias AlertNotification
	aux := &struct {
		Timestamp []interface{} `json:"timestamp"`
		*Alias
	}{
		Alias: (*Alias)(a),
	}

	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	if len(aux.Timestamp) == 7 {
		year, month, day := int(aux.Timestamp[0].(float64)), int(aux.Timestamp[1].(float64)), int(aux.Timestamp[2].(float64))
		hour, minute, second := int(aux.Timestamp[3].(float64)), int(aux.Timestamp[4].(float64)), int(aux.Timestamp[5].(float64))
		nanosecond := int(aux.Timestamp[6].(float64))

		// Construct the time using the extracted components.
		a.Timestamp = time.Date(year, time.Month(month), day, hour, minute, second, nanosecond, time.UTC)
	}

	return nil
}

// Consumer reads AlertNotification messages and enqueues tasks.
type Consumer struct {
	consumerGroup sarama.ConsumerGroup
	svc           Service
	logger        *logging.Logger
	mu            sync.Mutex
	lastSeen      map[string]time.Time
	ctx           context.Context
	cancel        context.CancelFunc
}

// Service defines dependencies needed by Consumer.
type Service interface {
	QueueTask(models.Task)
	Logger() *logging.Logger
}

// NewConsumer constructs a new Consumer.
func NewConsumer(brokers []string, topic, groupID string, svc Service) (*Consumer, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Configure consumer group
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = sarama.OffsetNewest
	config.Consumer.Group.Session.Timeout = 120 * time.Second   // Increase session timeout
	config.Consumer.Group.Heartbeat.Interval = 30 * time.Second // Increase heartbeat interval
	config.Consumer.MaxWaitTime = 2 * time.Second
	config.Consumer.Return.Errors = true
	sarama.Logger = log.New(os.Stdout, "", log.LstdFlags)

	// Tạo consumer group
	consumerGroup, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer group: %w", err)
	}

	svc.Logger().Infof("Starting consumer for topic %s with groupID %s", topic, groupID)
	svc.Logger().Debugf("Consumer group session timeout: %v, heartbeat interval: %v", config.Consumer.Group.Session.Timeout, config.Consumer.Group.Heartbeat.Interval)
	return &Consumer{
		consumerGroup: consumerGroup,
		svc:           svc,
		logger:        svc.Logger(),
		lastSeen:      make(map[string]time.Time),
		ctx:           ctx,
		cancel:        cancel,
	}, nil
}

// Start launches the consume loop in a goroutine.
func (c *Consumer) Start(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		c.logger.Infof("Kafka consumer started for topic %s", "alert_notification")
		for {
			// Consume messages
			err := c.consumerGroup.Consume(c.ctx, []string{"alert_notification"}, c)
			if err != nil {
				if c.ctx.Err() != nil {
					c.logger.Infof("Consumer context canceled, exiting")
					return
				}
				if err.Error() == "kafka server: A rebalance for the group is in progress. Please re-join the group" {
					c.logger.Warnf("Rebalance in progress, backing off for 5s")
					time.Sleep(5 * time.Second)
					continue
				}
				c.logger.Errorf("consume error: %v", err)
				time.Sleep(time.Second)
			}
		}
	}()
}

// Close stops consumption and closes the consumer.
func (c *Consumer) Close() error {
	c.cancel()
	if err := c.consumerGroup.Close(); err != nil {
		c.logger.Errorf("error closing consumer group: %v", err)
		return fmt.Errorf("close error: %w", err)
	}
	c.logger.Infof("Kafka consumer closed")
	return nil
}

// ConsumeClaim implements sarama.ConsumerGroupHandler.
func (c *Consumer) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for message := range claim.Messages() {
		start := time.Now()
		c.logger.Infof("Attempting to process message from partition %d at offset %d", message.Partition, message.Offset)

		t2 := time.Now()
		var alert AlertNotification
		if err := json.Unmarshal(message.Value, &alert); err != nil {
			c.logger.Errorf("unmarshal took %v, error: %v, raw message: %s", time.Since(t2), err, string(message.Value))
			session.MarkMessage(message, "")
			continue
		}
		c.logger.Debugf("unmarshal took %v", time.Since(t2))
		c.logger.Infof("Received alert %s at %s", alert.AlertID, alert.Timestamp)

		t3 := time.Now()
		c.mu.Lock()
		last, ok := c.lastSeen[alert.AlertID]
		if ok && !alert.Timestamp.After(last) {
			c.mu.Unlock()
			c.logger.Infof("outdated alert %s (seen %s)", alert.AlertID, last)
			session.MarkMessage(message, "")
			c.logger.Debugf("deduplication took %v", time.Since(t3))
			continue
		}
		c.lastSeen[alert.AlertID] = alert.Timestamp
		c.mu.Unlock()
		c.logger.Debugf("deduplication took %v", time.Since(t3))

		t4 := time.Now()
		task := models.Task{
			RequestID:    alert.AlertID,
			Subject:      alert.AlertName,
			Body:         alert.Message,
			RecipientID:  alert.UserID,
			Severity:     alert.Severity,
			TypeMessage:  alert.TypeMessage,
			Topic:        "alert_notification",
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
		c.logger.Debugf("queue task took %v", time.Since(t4))
		c.logger.Infof("Task queued for alert %s", alert.AlertID)

		session.MarkMessage(message, "")
		c.logger.Debugf("commit took %v", time.Since(t4))

		c.logger.Debugf("consumeNext took %v", time.Since(start))
	}

	return nil
}

// Setup is run at the beginning of a new session.
func (c *Consumer) Setup(_ sarama.ConsumerGroupSession) error {
	return nil
}

// Cleanup is run at the end of a session.
func (c *Consumer) Cleanup(_ sarama.ConsumerGroupSession) error {
	return nil
}
