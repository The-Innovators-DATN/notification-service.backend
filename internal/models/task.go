package models

import "time"

// Task represents an alert or services request enqueued for processing.
type Task struct {
	RequestID   string    // Unique identifier of the alert
	Subject     string    // Title or summary of the alert
	Body        string    // Detailed message content
	RecipientID int       // User ID to notify
	Severity    int       // Alert severity level
	TypeMessage string    // Current type of the alert (e.g., "alert", "resolved")
	Topic       string    // Source or category of the alert (e.g., Kafka topic)
	Timestamp   time.Time // When the alert event occurred
	Silenced    int

	// Contextual metric data
	StationID    int     // ID of related station or device
	MetricID     int     // ID of the metric
	MetricName   string  // Name of the metric
	Operator     string  // Comparison operator for threshold
	Threshold    float64 // Primary threshold value
	ThresholdMin float64 // Lower bound for range checks
	ThresholdMax float64 // Upper bound for range checks
	Value        float64 // Actual measured value
}
