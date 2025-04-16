package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"time"

	"notification-service/internal/models"
)

func (d *DB) CreateNotification(ctx context.Context, n models.Notification) error {
	query := `
        INSERT INTO notifications (
            id, created_at, type, subject, body, notification_policy_id, status, 
            recipient_id, request_id, latest_status, station_id, metric_id, 
            metric_name, operator, threshold, threshold_min, threshold_max, value
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)`
	_, err := d.Conn.Exec(ctx, query,
		n.ID, n.CreatedAt, n.Type, n.Subject, n.Body, n.NotificationPolicyID,
		n.Status, n.RecipientID, n.RequestID, n.LatestStatus, n.StationID,
		n.MetricID, n.MetricName, n.Operator, n.Threshold, n.ThresholdMin,
		n.ThresholdMax, n.Value)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (d *DB) UpdateNotificationStatus(ctx context.Context, requestID, status, lastError string) error {
	query := `
        UPDATE notifications
        SET status = $1, last_error = $2,
            sent_at = CASE WHEN $1 = 'sent' THEN $3 ELSE sent_at END
        WHERE request_id::text = $4`
	result, err := d.Conn.Exec(ctx, query, status, lastError, time.Now(), requestID)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no notification updated for request_id %s", requestID)
	}
	return nil
}

func (d *DB) GetLatestNotification(ctx context.Context, requestID string) (models.Notification, error) {
	var n models.Notification
	var id, policyID, reqID pgtype.UUID

	query := `
        SELECT id, created_at, sent_at, type, subject, body, notification_policy_id, status, 
               recipient_id, request_id, last_error, latest_status, station_id, metric_id, 
               metric_name, operator, threshold, threshold_min, threshold_max, value
        FROM notifications
        WHERE request_id::text = $1
        ORDER BY created_at DESC
        LIMIT 1`
	err := d.Conn.QueryRow(ctx, query, requestID).Scan(
		&id, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body,
		&policyID, &n.Status, &n.RecipientID, &reqID, &n.LastError, &n.LatestStatus,
		&n.StationID, &n.MetricID, &n.MetricName, &n.Operator, &n.Threshold,
		&n.ThresholdMin, &n.ThresholdMax, &n.Value,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return models.Notification{}, fmt.Errorf("no notification found for request_id %s", requestID)
		}
		return models.Notification{}, fmt.Errorf("failed to get latest notification for request_id %s: %w", requestID, err)
	}

	n.ID = id.Bytes
	n.NotificationPolicyID = policyID.Bytes
	n.RequestID = reqID.Bytes
	return n, nil
}

func (d *DB) GetSentNotificationsByUserID(ctx context.Context, userID int) ([]models.Notification, error) {
	rows, err := d.Conn.Query(ctx, `
        SELECT id, created_at, sent_at, type, subject, body, notification_policy_id, status, 
               recipient_id, request_id, last_error, latest_status, station_id, metric_id, 
               metric_name, operator, threshold, threshold_min, threshold_max, value
        FROM notifications
        WHERE recipient_id = $1 AND status = 'sent'
        ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get sent notifications by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var id, policyID, reqID pgtype.UUID
		err := rows.Scan(
			&id, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body,
			&policyID, &n.Status, &n.RecipientID, &reqID, &n.LastError, &n.LatestStatus,
			&n.StationID, &n.MetricID, &n.MetricName, &n.Operator, &n.Threshold,
			&n.ThresholdMin, &n.ThresholdMax, &n.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		n.ID = id.Bytes
		n.NotificationPolicyID = policyID.Bytes
		n.RequestID = reqID.Bytes
		notifications = append(notifications, n)
	}

	return notifications, nil
}

func (d *DB) GetAllNotifications(ctx context.Context, limit, offset int) ([]models.Notification, error) {
	query := `
        SELECT id, created_at, sent_at, type, subject, body, notification_policy_id, status, 
               recipient_id, request_id, last_error, latest_status, station_id, metric_id, 
               metric_name, operator, threshold, threshold_min, threshold_max, value
        FROM notifications
        ORDER BY created_at DESC
        LIMIT $1 OFFSET $2`
	rows, err := d.Conn.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get all notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var id, policyID, reqID pgtype.UUID
		err := rows.Scan(
			&id, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body,
			&policyID, &n.Status, &n.RecipientID, &reqID, &n.LastError, &n.LatestStatus,
			&n.StationID, &n.MetricID, &n.MetricName, &n.Operator, &n.Threshold,
			&n.ThresholdMin, &n.ThresholdMax, &n.Value,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan notification: %w", err)
		}
		n.ID = id.Bytes
		n.NotificationPolicyID = policyID.Bytes
		n.RequestID = reqID.Bytes
		notifications = append(notifications, n)
	}

	return notifications, nil
}
