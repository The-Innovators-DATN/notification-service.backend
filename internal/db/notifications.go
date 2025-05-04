package db

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreateNotification inserts a new notification record with nested AlertContext fields.
func (d *DB) CreateNotification(ctx context.Context, n models.Notification) error {
	if n.ID == [16]byte{} {
		newID := uuid.New()
		copy(n.ID[:], newID[:])
	}
	notifID := uuid.UUID(n.ID)
	policyFK := uuid.UUID(n.NotificationPolicyID)
	reqID := uuid.UUID(n.RequestID)

	query := `
	INSERT INTO notifications (
		id, created_at, type, subject, body,
		notification_policy_id, status, delivery_method,
		recipient_id, request_id, error, silenced,
		station_id, metric_id, metric_name, operator,
		threshold, threshold_min, threshold_max, value,
		updated_at
	)
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20)`

	_, err := d.Conn.Exec(ctx, query,
		notifID,
		n.CreatedAt,
		n.Type,
		n.Subject,
		n.Body,
		policyFK,
		n.Status,
		n.DeliveryMethod,
		n.RecipientID,
		reqID,
		n.Error,
		n.Silenced,
		n.Context.StationID,
		n.Context.MetricID,
		n.Context.MetricName,
		n.Context.Operator,
		n.Context.Threshold,
		n.Context.ThresholdMin,
		n.Context.ThresholdMax,
		n.Context.Value,
		n.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create notification: %w", err)
	}
	return nil
}

// UpdateNotificationStatus updates status and error by request ID.
func (d *DB) UpdateNotificationStatus(ctx context.Context, requestID string, status, errMsg string) error {
	reqID, err := uuid.Parse(requestID)
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
		return fmt.Errorf("no notification updated for request_id %s", requestID)
	}
	return nil
}

// GetNotificationsByUserID returns notifications with nested Policy and ContactPoint.
func (d *DB) GetNotificationsByUserID(ctx context.Context, userID, limit, offset int, statusFilter string) ([]models.Notification, int, error) {
	// Count total
	countQ := `SELECT COUNT(*) FROM notifications WHERE recipient_id = $1`
	countArgs := []interface{}{userID}
	if statusFilter != "all" {
		countQ += " AND status = $2"
		countArgs = append(countArgs, statusFilter)
	}

	var total int
	if err := d.Conn.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Query with LEFT JOINs
	query := `
	SELECT
		n.id, n.created_at, n.updated_at, n.type, n.subject, n.body,
		n.notification_policy_id, n.status, n.delivery_method,
		n.recipient_id, n.request_id, n.error, n.silenced,
		n.station_id, n.metric_id, n.metric_name, n.operator,
		n.threshold, n.threshold_min, n.threshold_max, n.value,
		p.id, p.severity, p.action, p.condition_type, p.contact_point_id,
		cp.id, cp.name, cp.type, cp.configuration
	FROM notifications n
	LEFT JOIN notification_policy p ON n.notification_policy_id = p.id AND p.status = 'active'
	LEFT JOIN contact_points cp     ON p.contact_point_id    = cp.id AND cp.status = 'active'
	WHERE n.recipient_id = $1`

	args := []interface{}{userID}
	if statusFilter != "all" {
		query += " AND n.status = $2"
		args = append(args, statusFilter)
		query += " ORDER BY n.created_at DESC LIMIT $3 OFFSET $4"
		args = append(args, limit, offset)
	} else {
		query += " ORDER BY n.created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	}

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var list []models.Notification
	for rows.Next() {
		var n models.Notification
		// nullable fields
		var errText sql.NullString
		var polID sql.NullString
		var polSeverity sql.NullInt64
		var polAction, polCond, polCPID sql.NullString
		var cpID sql.NullString
		var cpName, cpType sql.NullString
		var cpConfig map[string]interface{}

		err = rows.Scan(
			&n.ID, &n.CreatedAt, &n.UpdatedAt, &n.Type,
			&n.Subject, &n.Body,
			&n.NotificationPolicyID, &n.Status, &n.DeliveryMethod,
			&n.RecipientID, &n.RequestID, &errText,
			&n.Context.StationID, &n.Context.MetricID, &n.Context.MetricName, &n.Context.Operator,
			&n.Context.Threshold, &n.Context.ThresholdMin, &n.Context.ThresholdMax, &n.Context.Value,
			// policy
			&polID, &polSeverity, &polAction, &polCond, &polCPID,
			// contact point
			&cpID, &cpName, &cpType, &cpConfig,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if errText.Valid {
			n.Error = errText.String
		}

		// only attach Policy if present
		if polID.Valid {
			uid, _ := uuid.Parse(polID.String)
			var policyID [16]byte
			copy(policyID[:], uid[:])
			n.Policy = &models.Policy{
				ID:            policyID,
				Severity:      int(polSeverity.Int64),
				Action:        polAction.String,
				ConditionType: polCond.String,
				ContactPointID: func() [16]byte {
					if polCPID.Valid {
						id, _ := uuid.Parse(polCPID.String)
						var b [16]byte
						copy(b[:], id[:])
						return b
					}
					return [16]byte{}
				}(),
			}
		}

		// only attach ContactPoint if present
		if cpID.Valid {
			u, _ := uuid.Parse(cpID.String)
			var cpIDArr [16]byte
			copy(cpIDArr[:], u[:])
			n.ContactPoint = &models.ContactPoint{
				ID:            cpIDArr,
				Name:          cpName.String,
				Type:          cpType.String,
				Configuration: cpConfig,
			}
		}

		list = append(list, n)
	}

	return list, total, nil
}

// GetAllNotifications returns all notifications with nested Policy and ContactPoint, pagination.
func (d *DB) GetAllNotifications(ctx context.Context, statusFilter string, limit, offset int) ([]models.Notification, int, error) {
	// Count total
	countQ := `SELECT COUNT(*) FROM notifications`
	countArgs := []interface{}{}
	if statusFilter != "all" {
		countQ += " AND status = $1"
		countArgs = append(countArgs, statusFilter)
	}

	var total int
	if err := d.Conn.QueryRow(ctx, countQ, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count notifications: %w", err)
	}

	// Query with LEFT JOINs
	query := `
	SELECT
		n.id, n.created_at, n.updated_at, n.type, n.subject, n.body,
		n.notification_policy_id, n.status, n.delivery_method,
		n.recipient_id, n.request_id, n.error,
		n.station_id, n.metric_id, n.metric_name, n.operator,
		n.threshold, n.threshold_min, n.threshold_max, n.value,
		p.id, p.severity, p.action, p.condition_type, p.contact_point_id,
		cp.id, cp.name, cp.type, cp.configuration
	FROM notifications n
	LEFT JOIN notification_policy p ON n.notification_policy_id = p.id AND p.status = 'active'
	LEFT JOIN contact_points cp     ON p.contact_point_id    = cp.id AND cp.status = 'active'`

	args := []interface{}{}
	if statusFilter != "all" {
		query += " AND n.status = $1"
		args = append(args, statusFilter)
		query += " ORDER BY n.created_at DESC LIMIT $2 OFFSET $3"
		args = append(args, limit, offset)
	} else {
		query += " ORDER BY n.created_at DESC LIMIT $1 OFFSET $2"
		args = append(args, limit, offset)
	}

	rows, err := d.Conn.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notifications: %w", err)
	}
	defer rows.Close()

	var list []models.Notification
	for rows.Next() {
		var n models.Notification
		// nullable fields
		var errText sql.NullString
		var polID sql.NullString
		var polSeverity sql.NullInt64
		var polAction, polCond, polCPID sql.NullString
		var cpID sql.NullString
		var cpName, cpType sql.NullString
		var cpConfig map[string]interface{}

		err = rows.Scan(
			&n.ID, &n.CreatedAt, &n.UpdatedAt, &n.Type,
			&n.Subject, &n.Body,
			&n.NotificationPolicyID, &n.Status, &n.DeliveryMethod,
			&n.RecipientID, &n.RequestID, &errText,
			&n.Context.StationID, &n.Context.MetricID, &n.Context.MetricName, &n.Context.Operator,
			&n.Context.Threshold, &n.Context.ThresholdMin, &n.Context.ThresholdMax, &n.Context.Value,
			// policy
			&polID, &polSeverity, &polAction, &polCond, &polCPID,
			// contact point
			&cpID, &cpName, &cpType, &cpConfig,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan notification: %w", err)
		}

		if errText.Valid {
			n.Error = errText.String
		}

		// only attach Policy if present
		if polID.Valid {
			uid, _ := uuid.Parse(polID.String)
			var policyID [16]byte
			copy(policyID[:], uid[:])
			n.Policy = &models.Policy{
				ID:            policyID,
				Severity:      int(polSeverity.Int64),
				Action:        polAction.String,
				ConditionType: polCond.String,
				ContactPointID: func() [16]byte {
					if polCPID.Valid {
						id, _ := uuid.Parse(polCPID.String)
						var b [16]byte
						copy(b[:], id[:])
						return b
					}
					return [16]byte{}
				}(),
			}
		}

		// only attach ContactPoint if present
		if cpID.Valid {
			u, _ := uuid.Parse(cpID.String)
			var cpIDArr [16]byte
			copy(cpIDArr[:], u[:])
			n.ContactPoint = &models.ContactPoint{
				ID:            cpIDArr,
				Name:          cpName.String,
				Type:          cpType.String,
				Configuration: cpConfig,
			}
		}

		list = append(list, n)
	}

	return list, total, nil
}
