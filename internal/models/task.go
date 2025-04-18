package models

import "time"

type Task struct {
	RequestID    string
	Subject      string
	Body         string
	RecipientID  int
	Severity     int
	Status       string
	Topic        string
	PolicyID     string
	Timestamp    time.Time
	StationID    int
	MetricID     int
	MetricName   string
	Operator     string
	Threshold    float64
	ThresholdMin float64
	ThresholdMax float64
	Value        float64
}
