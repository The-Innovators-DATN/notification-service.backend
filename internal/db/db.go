package db

import (
	"context"
	"github.com/jackc/pgx/v5"
	"notification-service/internal/models"
	"time"
)

type DB struct {
	Conn *pgx.Conn
}

func New(dsn string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, err
	}
	return &DB{Conn: conn}, nil
}

func (d *DB) Close() error {
	return d.Conn.Close(context.Background())
}

func (d *DB) CreateNotification(ctx context.Context, n models.Notification) error {
	_, err := d.Conn.Exec(ctx, `
		INSERT INTO notifications (id, created_at, type, subject, body, notification_policy_id, status, recipient_id, request_id, retry_count, latest_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (request_id) DO NOTHING`,
		n.ID, n.CreatedAt, n.Type, n.Subject, n.Body, n.NotificationPolicyID, n.Status, n.RecipientID, n.RequestID, n.RetryCount, n.LatestStatus)
	return err
}

func (d *DB) UpdateNotificationStatus(ctx context.Context, requestID, status, lastError string) error {
	_, err := d.Conn.Exec(ctx, `
		UPDATE notifications 
		SET status = $1, last_error = $2, retry_count = retry_count + 1, 
			sent_at = CASE WHEN $1 = 'sent' THEN $3 ELSE sent_at END
		WHERE request_id = $4 AND status != 'sent' AND status != 'cancelled'`,
		status, lastError, time.Now(), requestID)
	return err
}

func (d *DB) GetLatestStatus(ctx context.Context, requestID string) (string, error) {
	var status string
	err := d.Conn.QueryRow(ctx, `SELECT latest_status FROM notifications WHERE request_id = $1`, requestID).Scan(&status)
	return status, err
}

func (d *DB) GetPolicyAndContact(ctx context.Context, topic string, severity int) (models.Policy, models.ContactPoint, error) {
	var p models.Policy
	var cp models.ContactPoint
	err := d.Conn.QueryRow(ctx, `
		SELECT p.id, p.contact_point_id, p.severity, p.status, p.topic, p.created_at,
		       cp.id, cp.name, cp.user_id, cp.type, cp.configuration, cp.status, cp.created_at
		FROM notification_policy p
		JOIN contact_points cp ON p.contact_point_id = cp.id
		WHERE p.topic = $1 AND p.severity <= $2 AND p.status = 'active'
		ORDER BY p.severity DESC
		LIMIT 1`, topic, severity).Scan(
		&p.ID, &p.ContactPointID, &p.Severity, &p.Status, &p.Topic, &p.CreatedAt,
		&cp.ID, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt)
	return p, cp, err
}

func (d *DB) GetAllNotifications(ctx context.Context) ([]models.Notification, error) {
	rows, err := d.Conn.Query(ctx, `SELECT id, created_at, sent_at, type, subject, body, notification_policy_id, status, recipient_id, request_id, retry_count, last_error, latest_status FROM notifications`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []models.Notification
	for rows.Next() {
		var n models.Notification
		if err := rows.Scan(&n.ID, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body, &n.NotificationPolicyID, &n.Status, &n.RecipientID, &n.RequestID, &n.RetryCount, &n.LastError, &n.LatestStatus); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, nil
}

func (d *DB) GetNotificationByID(ctx context.Context, id string) (models.Notification, error) {
	var n models.Notification
	err := d.Conn.QueryRow(ctx, `SELECT id, created_at, sent_at, type, subject, body, notification_policy_id, status, recipient_id, request_id, retry_count, last_error, latest_status FROM notifications WHERE id = $1`, id).
		Scan(&n.ID, &n.CreatedAt, &n.SentAt, &n.Type, &n.Subject, &n.Body, &n.NotificationPolicyID, &n.Status, &n.RecipientID, &n.RequestID, &n.RetryCount, &n.LastError, &n.LatestStatus)
	return n, err
}

func (d *DB) CreateContactPoint(ctx context.Context, cp models.ContactPoint) error {
	_, err := d.Conn.Exec(ctx, `
		INSERT INTO contact_points (id, name, user_id, type, configuration, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		SET name = $2, user_id = $3, type = $4, configuration = $5, status = $6`,
		cp.ID, cp.Name, cp.UserID, cp.Type, cp.Configuration, cp.Status, cp.CreatedAt)
	return err
}

func (d *DB) GetContactPoint(ctx context.Context, id string) (models.ContactPoint, error) {
	var cp models.ContactPoint
	err := d.Conn.QueryRow(ctx, `
		SELECT id, name, user_id, type, configuration, status, created_at
		FROM contact_points WHERE id = $1`, id).
		Scan(&cp.ID, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt)
	return cp, err
}

func (d *DB) CreatePolicy(ctx context.Context, p models.Policy) error {
	_, err := d.Conn.Exec(ctx, `
		INSERT INTO notification_policy (id, contact_point_id, severity, status, topic, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET contact_point_id = $2, severity = $3, status = $4, topic = $5`,
		p.ID, p.ContactPointID, p.Severity, p.Status, p.Topic, p.CreatedAt)
	return err
}
