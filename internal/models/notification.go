package models

import "time"

type Notification struct {
	ID                   [16]byte  `json:"id"`
	CreatedAt            time.Time `json:"created_at"`
	Type                 string    `json:"type"`
	Subject              string    `json:"subject"`
	Body                 string    `json:"body"`
	NotificationPolicyID [16]byte  `json:"notification_policy_id"`
	Status               string    `json:"status"`
	DeliveryMethod       string    `json:"delivery_method"`
	RecipientID          int64     `json:"recipient_id"`
	RequestID            [16]byte  `json:"request_id"`
	Error                string    `json:"error"`
}
