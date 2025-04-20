package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"notification-service/internal/models"
)

func (d *DB) CreateNotification(ctx context.Context, n models.Notification) error {
	query := `
        INSERT INTO notifications (
            id, created_at, type, subject, body, notification_policy_id, status, 
            delivery_method, recipient_id, request_id, error
        )
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`
	_, err := d.Conn.Exec(ctx, query,
		n.ID, n.CreatedAt, n.Type, n.Subject, n.Body, n.NotificationPolicyID,
		n.Status, n.DeliveryMethod, n.RecipientID, n.RequestID, n.Error)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

func (d *DB) UpdateNotificationStatus(ctx context.Context, requestID, status, error string) error {
	query := `
        UPDATE notifications
        SET status = $1, error = $2,
        WHERE request_id::text = $3`
	result, err := d.Conn.Exec(ctx, query, status, error, requestID)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}
	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no notification updated for request_id %s", requestID)
	}
	return nil
}

//func (d *DB) GetLatestNotification(ctx context.Context, requestID string) (models.Notification, error) {
//	var n models.Notification
//	var id, policyID, reqID pgtype.UUID
//
//	query := `
//        SELECT id, created_at, type, subject, body, notification_policy_id, status,
//               recipient_id, request_id, last_error, latest_status, station_id, metric_id,
//               metric_name, operator, threshold, threshold_min, threshold_max, value
//        FROM notifications
//        WHERE request_id::text = $1
//        ORDER BY created_at DESC
//        LIMIT 1`
//	err := d.Conn.QueryRow(ctx, query, requestID).Scan(
//		&id, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body,
//		&policyID, &n.Status, &n.RecipientID, &reqID, &n.LastError, &n.LatestStatus,
//		&n.StationID, &n.MetricID, &n.MetricName, &n.Operator, &n.Threshold,
//		&n.ThresholdMin, &n.ThresholdMax, &n.Value,
//	)
//	if err != nil {
//		if err == pgx.ErrNoRows {
//			return models.Notification{}, fmt.Errorf("no notification found for request_id %s", requestID)
//		}
//		return models.Notification{}, fmt.Errorf("failed to get latest notification for request_id %s: %w", requestID, err)
//	}
//
//	n.ID = id.Bytes
//	n.NotificationPolicyID = policyID.Bytes
//	n.RequestID = reqID.Bytes
//	return n, nil
//}

func (d *DB) GetNotificationsByUserID(ctx context.Context, userID, limit, offset int, status string) ([]models.Notification, int, error) {
	var (
		query   string
		args    []interface{}
		counter int
	)

	// Base query and args
	query = `
        SELECT id, created_at, type, subject, body, notification_policy_id, status, 
               delivery_method, recipient_id, request_id, error
        FROM notifications
        WHERE recipient_id = $1`
	args = append(args, userID)

	countQuery := "SELECT COUNT(*) FROM notifications WHERE recipient_id = $1"
	countArgs := []interface{}{userID}

	// Apply status filter
	if status != "all" {
		query += " AND status = $2"
		countQuery += " AND status = $2"
		countArgs = append(countArgs, status)
		args = append(args, status)
	} else {
		args = append(args, limit, offset)
	}

	err := d.Conn.QueryRow(ctx, countQuery, countArgs...).Scan(&counter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Add ordering and pagination
	query += " ORDER BY created_at DESC LIMIT $3 OFFSET $4"

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications by user_id %d and status %s: %w", userID, status, err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var id, policyID, requestID pgtype.UUID
		var errorText pgtype.Text

		err := rows.Scan(
			&id, &n.CreatedAt, &n.Type, &n.Subject, &n.Body,
			&policyID, &n.Status, &n.DeliveryMethod, &n.RecipientID,
			&requestID, &errorText,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if id.Valid {
			n.ID = id.Bytes
		}
		if policyID.Valid {
			n.NotificationPolicyID = policyID.Bytes
		}
		if requestID.Valid {
			n.RequestID = requestID.Bytes
		}
		if errorText.Valid {
			n.Error = errorText.String
		}

		notifications = append(notifications, n)
	}

	return notifications, counter, nil
}

func (d *DB) GetAllNotifications(ctx context.Context, status string, limit, offset int) ([]models.Notification, int, error) {
	var (
		query   string
		args    []interface{}
		counter int
	)

	query = `
        SELECT id, created_at, type, subject, body, notification_policy_id, status, 
               delivery_method, recipient_id, request_id, error
		FROM notifications`
	args = []interface{}{}
	countQuery := `SELECT COUNT(*) FROM notifications`
	countArgs := []interface{}{}

	if status != "all" {
		query += " WHERE status = $3"
		countQuery += " WHERE status = $1"
		args = append(args, limit, offset, status)
		countArgs = append(countArgs, status)
	} else {
		args = append(args, limit, offset)
	}

	// Count total notifications
	err := d.Conn.QueryRow(ctx, countQuery, countArgs...).Scan(&counter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count all notifications: %w", err)
	}

	query += " ORDER BY created_at DESC LIMIT $1 OFFSET $2"

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all notifications: %w", err)
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		var id, policyID, requestID pgtype.UUID
		var errorText pgtype.Text

		err := rows.Scan(
			&id, &n.CreatedAt, &n.Type, &n.Subject, &n.Body,
			&policyID, &n.Status, &n.DeliveryMethod, &n.RecipientID,
			&requestID, &errorText,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if id.Valid {
			n.ID = id.Bytes
		}
		if policyID.Valid {
			n.NotificationPolicyID = policyID.Bytes
		}
		if requestID.Valid {
			n.RequestID = requestID.Bytes
		}
		if errorText.Valid {
			n.Error = errorText.String
		}

		notifications = append(notifications, n)
	}

	return notifications, counter, nil
}
