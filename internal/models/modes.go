package models

import (
	"time"
)

type Notification struct {
	ID                   string     `json:"id"`
	CreatedAt            time.Time  `json:"created_at"`
	SentAt               *time.Time `json:"sent_at"`
	Type                 string     `json:"type"`
	Subject              string     `json:"subject"`
	Body                 string     `json:"body"`
	NotificationPolicyID string     `json:"notification_policy_id"`
	Status               string     `json:"status"`
	RecipientID          int        `json:"recipient_id"`
	RequestID            string     `json:"request_id"`
	RetryCount           int        `json:"retry_count"`
	LastError            string     `json:"last_error"`
	LatestStatus         string     `json:"latest_status"`
}

type ContactPoint struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	UserID        int                    `json:"user_id"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"`
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
}

type Policy struct {
	ID             string    `json:"id"`
	ContactPointID string    `json:"contact_point_id"`
	Severity       int       `json:"severity"`
	Status         string    `json:"status"`
	Topic          string    `json:"topic"`
	CreatedAt      time.Time `json:"created_at"`
}
