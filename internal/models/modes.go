package models

import "time"

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
}

type ContactPoint struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	UserID        int64                  `json:"user_id"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
}

type Policy struct {
	ID             string    `json:"id"`
	ContactPointID string    `json:"contact_point_id"`
	Severity       int16     `json:"severity"`
	Status         string    `json:"status"`
	Topic          string    `json:"topic"`
	CreatedAt      time.Time `json:"created_at"`
}

type Notification struct {
	ID                   string    `json:"id"`
	CreatedAt            time.Time `json:"created_at"`
	SentAt               time.Time `json:"sent_at"`
	Type                 string    `json:"type"`
	Subject              string    `json:"subject"`
	Body                 string    `json:"body"`
	NotificationPolicyID string    `json:"notification_policy_id"`
	Status               string    `json:"status"`
	RecipientID          int       `json:"recipient_id"`
	RequestID            string    `json:"request_id"`
	RetryCount           int       `json:"retry_count"`
	LastError            string    `json:"last_error"`
	LatestStatus         string    `json:"latest_status"`
}
