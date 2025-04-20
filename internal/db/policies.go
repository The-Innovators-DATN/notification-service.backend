package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"notification-service/internal/models"
)

func (d *DB) CreatePolicy(ctx context.Context, p models.Policy) error {
	query := `
		INSERT INTO notification_policy (id, contact_point_id, severity, status, action, created_at, updated_at, condition_type)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6)
		ON CONFLICT (id) DO UPDATE
		SET contact_point_id = $2, severity = $3, status = $4, action = $5, updated_at = NOW(), condition_type = $6`
	_, err := d.Conn.Exec(ctx, query,
		p.ID, p.ContactPointID, p.Severity, p.Status, p.Action, p.ConditionType)
	if err != nil {
		return fmt.Errorf("failed to create policy: %w", err)
	}
	return nil
}

func (d *DB) GetPolicyById(ctx context.Context, id string) (models.Policy, error) {
	var p models.Policy
	var uuid, cpUUID pgtype.UUID
	query := `
		SELECT id, contact_point_id, severity, status, action, created_at, updated_at, condition_type
		FROM notification_policy
		WHERE id::text = $1 AND status = 'active'`
	err := d.Conn.QueryRow(ctx, query, id).Scan(
		&uuid, &cpUUID, &p.Severity, &p.Status, &p.Action, &p.CreatedAt, &p.UpdatedAt, &p.ConditionType,
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
		SELECT np.id, np.contact_point_id, np.severity, np.status, np.action, np.created_at, np.updated_at, np.condition_type
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
			&uuid, &cpUUID, &p.Severity, &p.Status, &p.Action, &p.CreatedAt, &p.UpdatedAt, &p.ConditionType,
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
		UPDATE notification_policy
		SET status = 'inactive', updated_at = NOW()
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
		SET contact_point_id = $1, severity = $2, status = $3, action = $4, updated_at = NOW(), condition_type = $5
		WHERE id::text = $6 AND status = 'active'`
	_, err := d.Conn.Exec(ctx, query,
		p.ContactPointID, p.Severity, p.Status, p.Action, p.ConditionType, p.ID)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}
