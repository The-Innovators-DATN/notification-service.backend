package models

import "time"

// Notification đại diện cho một thông báo trong database
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
	StationID            int     // station_id
	MetricID             int     // metric_id
	MetricName           string  // metric_name
	Operator             string  // operator
	Threshold            float64 // threshold
	ThresholdMin         float64 // threshold_min
	ThresholdMax         float64 // threshold_max
	Value                float64 // value
}
