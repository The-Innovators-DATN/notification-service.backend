package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

// AlertContext holds contextual alert details pulled from metrics.
type AlertContext struct {
	StationID    int     `json:"station_id,omitempty"`
	MetricID     int     `json:"metric_id,omitempty"`
	MetricName   string  `json:"metric_name,omitempty"`
	Operator     string  `json:"operator,omitempty"`
	Threshold    float64 `json:"threshold,omitempty"`
	ThresholdMin float64 `json:"threshold_min,omitempty"`
	ThresholdMax float64 `json:"threshold_max,omitempty"`
	Value        float64 `json:"value,omitempty"`
}

// Notification represents a delivered notification with context and error details.
type Notification struct {
	ID                   [16]byte      `json:"id"`
	CreatedAt            time.Time     `json:"created_at"`
	UpdatedAt            time.Time     `json:"updated_at"`
	Type                 string        `json:"type"`
	Subject              string        `json:"subject"`
	Body                 string        `json:"body"`
	NotificationPolicyID [16]byte      `json:"notification_policy_id"`
	Silenced             int           `json:"silenced"`
	Status               string        `json:"status"`
	DeliveryMethod       string        `json:"delivery_method"`
	RecipientID          int64         `json:"recipient_id"`
	RequestID            [16]byte      `json:"request_id"`
	Error                string        `json:"error,omitempty"`
	Context              AlertContext  `json:"context,omitempty"`
	Policy               *Policy       `json:"policy,omitempty"`        // Added for response, not stored in DB
	ContactPoint         *ContactPoint `json:"contact_point,omitempty"` // Added for response, not stored in DB
}

// MarshalJSON customizes JSON serialization for Notification to return UUIDs as strings.
func (n Notification) MarshalJSON() ([]byte, error) {
	type Alias Notification
	return json.Marshal(&struct {
		ID                   string `json:"id"`
		NotificationPolicyID string `json:"notification_policy_id"`
		RequestID            string `json:"request_id"`
		*Alias
	}{
		ID:                   uuid.UUID(n.ID).String(),
		NotificationPolicyID: uuid.UUID(n.NotificationPolicyID).String(),
		RequestID:            uuid.UUID(n.RequestID).String(),
		Alias:                (*Alias)(&n),
	})
}
