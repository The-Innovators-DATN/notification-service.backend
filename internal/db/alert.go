package db

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreateAlert inserts a new alert record into the database.
// It generates a new UUID for the uid column if not provided.
func (d *DB) CreateAlert(ctx context.Context, alert models.Task) error {
	// Generate uid if not set
	newUID := uuid.New()

	query := `
    INSERT INTO alert (
        uid, request_id, subject, body, recipient_id, severity, type_message, topic, timestamp, silenced,
        station_id, metric_id, metric_name, operator, threshold, threshold_min, threshold_max, value
    ) VALUES (
        $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
        $11, $12, $13, $14, $15, $16, $17, $18
    )`

	_, err := d.Pool.Exec(ctx, query,
		newUID,
		alert.RequestID,
		alert.Subject,
		alert.Body,
		alert.RecipientID,
		alert.Severity,
		alert.TypeMessage,
		alert.Topic,
		alert.Timestamp,
		alert.Silenced,
		alert.StationID,
		alert.MetricID,
		alert.MetricName,
		alert.Operator,
		alert.Threshold,
		alert.ThresholdMin,
		alert.ThresholdMax,
		alert.Value,
	)
	if err != nil {
		return fmt.Errorf("failed to insert alert: %w", err)
	}
	return nil
}

// GetAlertsByUserID fetches alerts for a given user with pagination and optional silenced filter.
func (d *DB) GetAlertsByUserID(ctx context.Context, userID, limit, offset int, silencedFilter string) ([]models.Task, int, error) {
	countQ := `SELECT COUNT(*) FROM alert WHERE recipient_id = $1`
	countArgs := []interface{}{userID}
	if silencedFilter != "all" {
		countQ += " AND silenced = $2"
		countArgs = append(countArgs, silencedFilter)
	}

	var total int
	if err := d.Pool.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	query := `
	SELECT
		request_id, subject, body, recipient_id, severity, type_message, topic, timestamp, silenced,
		station_id, metric_id, metric_name, operator, threshold, threshold_min, threshold_max, value
	FROM alert
	WHERE recipient_id = $1`

	args := []interface{}{userID}
	if silencedFilter != "all" {
		query += " AND silenced = $2 ORDER BY timestamp DESC LIMIT $3 OFFSET $4"
		args = append(args, silencedFilter, limit, offset)
	} else {
		query += " ORDER BY timestamp DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := d.Pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get alerts: %w", err)
	}
	defer rows.Close()

	var list []models.Task
	for rows.Next() {
		var alert models.Task
		err := rows.Scan(
			&alert.RequestID,
			&alert.Subject,
			&alert.Body,
			&alert.RecipientID,
			&alert.Severity,
			&alert.TypeMessage,
			&alert.Topic,
			&alert.Timestamp,
			&alert.Silenced,
			&alert.StationID,
			&alert.MetricID,
			&alert.MetricName,
			&alert.Operator,
			&alert.Threshold,
			&alert.ThresholdMin,
			&alert.ThresholdMax,
			&alert.Value,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan alert: %w", err)
		}
		list = append(list, alert)
	}

	return list, total, nil
}
