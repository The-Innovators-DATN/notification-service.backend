package db

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreateNotification inserts a new notification record with nested AlertContext fields.
func (d *DB) CreateNotification(ctx context.Context, n models.Notification) error {
	// Ensure ID is set
	if n.ID == [16]byte{} {
		newID := uuid.New()
		copy(n.ID[:], newID[:])
	}
	// Bind UUIDs
	notifID := uuid.UUID(n.ID)
	policyID := uuid.UUID(n.NotificationPolicyID)
	reqID := uuid.UUID(n.RequestID)

	query := `
	INSERT INTO notifications (
		id, created_at, type, subject, body,
		notification_policy_id, status, delivery_method,
		recipient_id, request_id, error,
		station_id, metric_id, metric_name, operator,
		threshold, threshold_min, threshold_max, value
	)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
	       $12, $13, $14, $15, $16, $17, $18, $19)`

	_, err := d.Conn.Exec(ctx, query,
		notifID,
		n.CreatedAt,
		n.Type,
		n.Subject,
		n.Body,
		policyID,
		n.Status,
		n.DeliveryMethod,
		n.RecipientID,
		reqID,
		n.Error,
		n.Context.StationID,
		n.Context.MetricID,
		n.Context.MetricName,
		n.Context.Operator,
		n.Context.Threshold,
		n.Context.ThresholdMin,
		n.Context.ThresholdMax,
		n.Context.Value,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// UpdateNotificationStatus updates status and error by request ID.
func (d *DB) UpdateNotificationStatus(ctx context.Context, requestIDStr, status, errMsg string) error {
	reqID, err := uuid.Parse(requestIDStr)
	if err != nil {
		return fmt.Errorf("invalid request_id UUID: %w", err)
	}

	query := `
	UPDATE notifications
	SET status = $1,
	    error = $2,
	    updated_at = NOW()
	WHERE request_id = $3`

	res, err := d.Conn.Exec(ctx, query, status, errMsg, reqID)
	if err != nil {
		return fmt.Errorf("failed to update notification status: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("no notification updated for request_id %s", requestIDStr)
	}
	return nil
}

// GetNotificationsByUserID returns a paginated list of notifications including AlertContext.
func (d *DB) GetNotificationsByUserID(ctx context.Context, userID, limit, offset int, statusFilter string) ([]models.Notification, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM notifications WHERE recipient_id = $1`
	countArgs := []interface{}{userID}
	if statusFilter != "all" {
		countQuery += " AND status = $2"
		countArgs = append(countArgs, statusFilter)
	}

	var total int
	err := d.Conn.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Fetch rows
	query := `
	SELECT id, created_at, type, subject, body,
	       notification_policy_id, status, delivery_method,
	       recipient_id, request_id, error,
	       station_id, metric_id, metric_name, operator,
	       threshold, threshold_min, threshold_max, value
	FROM notifications
	WHERE recipient_id = $1`
	args := []interface{}{userID}
	if statusFilter != "all" {
		query += " AND status = $2"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at DESC LIMIT $3 OFFSET $4"
	args = append(args, limit, offset)

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var list []models.Notification
	for rows.Next() {
		var n models.Notification
		var idUUID, policyUUID, reqUUID uuid.UUID
		var errText *string

		err = rows.Scan(
			&idUUID,
			&n.CreatedAt,
			&n.Type,
			&n.Subject,
			&n.Body,
			&policyUUID,
			&n.Status,
			&n.DeliveryMethod,
			&n.RecipientID,
			&reqUUID,
			&errText,
			&n.Context.StationID,
			&n.Context.MetricID,
			&n.Context.MetricName,
			&n.Context.Operator,
			&n.Context.Threshold,
			&n.Context.ThresholdMin,
			&n.Context.ThresholdMax,
			&n.Context.Value,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		// Copy IDs
		copy(n.ID[:], idUUID[:])
		copy(n.NotificationPolicyID[:], policyUUID[:])
		copy(n.RequestID[:], reqUUID[:])
		if errText != nil {
			n.Error = *errText
		}
		list = append(list, n)
	}

	return list, total, nil
}

// GetAllNotifications returns all notifications with optional status filter and pagination.
func (d *DB) GetAllNotifications(ctx context.Context, statusFilter string, limit, offset int) ([]models.Notification, int, error) {
	// Count total
	countQuery := `SELECT COUNT(*) FROM notifications`
	countArgs := []interface{}{}
	if statusFilter != "all" {
		countQuery += " WHERE status = $1"
		countArgs = append(countArgs, statusFilter)
	}

	var total int
	err := d.Conn.QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Fetch rows
	query := `
	SELECT id, created_at, type, subject, body,
	       notification_policy_id, status, delivery_method,
	       recipient_id, request_id, error,
	       station_id, metric_id, metric_name, operator,
	       threshold, threshold_min, threshold_max, value
	FROM notifications`
	args := []interface{}{}
	if statusFilter != "all" {
		query += " WHERE status = $1"
		args = append(args, statusFilter)
	}
	query += " ORDER BY created_at DESC LIMIT $2 OFFSET $3"
	args = append(args, limit, offset)

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get all notifications: %w", err)
	}
	defer rows.Close()

	var list []models.Notification
	for rows.Next() {
		var n models.Notification
		var idUUID, policyUUID, reqUUID uuid.UUID
		var errText *string

		err = rows.Scan(
			&idUUID,
			&n.CreatedAt,
			&n.Type,
			&n.Subject,
			&n.Body,
			&policyUUID,
			&n.Status,
			&n.DeliveryMethod,
			&n.RecipientID,
			&reqUUID,
			&errText,
			&n.Context.StationID,
			&n.Context.MetricID,
			&n.Context.MetricName,
			&n.Context.Operator,
			&n.Context.Threshold,
			&n.Context.ThresholdMin,
			&n.Context.ThresholdMax,
			&n.Context.Value,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}
		copy(n.ID[:], idUUID[:])
		copy(n.NotificationPolicyID[:], policyUUID[:])
		copy(n.RequestID[:], reqUUID[:])
		if errText != nil {
			n.Error = *errText
		}
		list = append(list, n)
	}

	return list, total, nil
}
