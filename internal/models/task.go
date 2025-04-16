package models

import "time"

// Task đại diện cho một công việc gửi thông báo
type Task struct {
	RequestID    string
	Subject      string
	Body         string
	RecipientID  int
	Severity     int
	Status       string
	Topic        string
	Timestamp    time.Time
	StationID    int     // station_id
	MetricID     int     // metric_id
	MetricName   string  // metric_name
	Operator     string  // operator
	Threshold    float64 // threshold
	ThresholdMin float64 // threshold_min
	ThresholdMax float64 // threshold_max
	Value        float64 // value
}
