package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"notification-service/internal/models"
)

func (d *DB) CreatePolicy(ctx context.Context, p models.Policy) error {
	query := `
		INSERT INTO notification_policy (id, contact_point_id, severity, status, topic, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (id) DO UPDATE
		SET contact_point_id = $2, severity = $3, status = $4, topic = $5`
	_, err := d.Conn.Exec(ctx, query,
		p.ID, p.ContactPointID, p.Severity, p.Status, p.Topic, p.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	return nil
}

func (d *DB) GetPolicy(ctx context.Context, id string) (models.Policy, error) {
	var p models.Policy
	var uuid, cpUUID pgtype.UUID
	query := `
		SELECT id, contact_point_id, severity, status, topic, created_at
		FROM notification_policy
		WHERE id::text = $1`
	err := d.Conn.QueryRow(ctx, query, id).Scan(
		&uuid, &cpUUID, &p.Severity, &p.Status, &p.Topic, &p.CreatedAt,
	)
	if err != nil {
		return models.Policy{}, fmt.Errorf("failed to get policy: %w", err)
	}
	p.ID = uuid.Bytes
	p.ContactPointID = cpUUID.Bytes
	return p, nil
}

func (d *DB) GetPoliciesByUserID(ctx context.Context, userID int64) ([]models.Policy, error) {
	rows, err := d.Conn.Query(ctx, `
		SELECT np.id, np.contact_point_id, np.severity, np.status, np.topic, np.created_at
		FROM notification_policy np
		JOIN contact_points cp ON np.contact_point_id = cp.id
		WHERE cp.user_id = $1 AND np.status = 'active'`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		var uuid, cpUUID pgtype.UUID
		err := rows.Scan(
			&uuid, &cpUUID, &p.Severity, &p.Status, &p.Topic, &p.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		p.ID = uuid.Bytes
		p.ContactPointID = cpUUID.Bytes
		policies = append(policies, p)
	}

	return policies, nil
}

func (d *DB) DeletePolicy(ctx context.Context, id string) error {
	query := `
		DELETE FROM notification_policy
		WHERE id::text = $1`
	_, err := d.Conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	return nil
}

func (d *DB) UpdatePolicy(ctx context.Context, p models.Policy) error {
	query := `
		UPDATE notification_policy
		SET contact_point_id = $1, severity = $2, status = $3, topic = $4
		WHERE id::text = $5`
	_, err := d.Conn.Exec(ctx, query,
		p.ContactPointID, p.Severity, p.Status, p.Topic, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}
