package models

import "time"

type Notification struct {
	ID                   [16]byte
	CreatedAt            time.Time
	SentAt               time.Time
	Type                 string
	Subject              string
	Body                 string
	NotificationPolicyID [16]byte
	Status               string
	RecipientID          int
	RequestID            [16]byte
	LastError            string
	LatestStatus         string
	StationID            int
	MetricID             int
	MetricName           string
	Operator             string
	Threshold            float64
	ThresholdMin         float64
	ThresholdMax         float64
	Value                float64
}
