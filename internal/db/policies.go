package db

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreatePolicy inserts or updates a notification policy record.
func (d *DB) CreatePolicy(ctx context.Context, p models.Policy) error {
	// Ensure ID is set
	if p.ID == [16]byte{} {
		newID := uuid.New()
		copy(p.ID[:], newID[:])
	}
	// Validate contact point ID
	if p.ContactPointID == [16]byte{} {
		return fmt.Errorf("contact point ID cannot be empty")
	}

	// Bind UUIDs
	policyID := uuid.UUID(p.ID)
	contactID := uuid.UUID(p.ContactPointID)

	query := `
	INSERT INTO notification_policy (
		id, contact_point_id, severity, status, action, created_at, updated_at, condition_type
	)
	VALUES ($1, $2, $3, $4, $5, NOW(), NOW(), $6)
	ON CONFLICT (id) DO UPDATE
	SET contact_point_id = EXCLUDED.contact_point_id,
	    severity = EXCLUDED.severity,
	    status = EXCLUDED.status,
	    action = EXCLUDED.action,
	    condition_type = EXCLUDED.condition_type,
	    updated_at = NOW()`

	_, err := d.Conn.Exec(ctx, query,
		policyID,
		contactID,
		p.Severity,
		p.Status,
		p.Action,
		p.ConditionType,
	)
	if err != nil {
		return fmt.Errorf("failed to create or update policy: %w", err)
	}
	return nil
}

// GetPolicyByID retrieves an active policy by its UUID string.
func (d *DB) GetPolicyByID(ctx context.Context, idStr string) (models.Policy, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return models.Policy{}, fmt.Errorf("invalid policy ID: %w", err)
	}

	query := `
	SELECT id, contact_point_id, severity, status, action, created_at, updated_at, condition_type
	FROM notification_policy
	WHERE id = $1 AND status = 'active'`

	var p models.Policy
	var fetchedID, contactID uuid.UUID
	err = d.Conn.QueryRow(ctx, query, id).Scan(
		&fetchedID,
		&contactID,
		&p.Severity,
		&p.Status,
		&p.Action,
		&p.CreatedAt,
		&p.UpdatedAt,
		&p.ConditionType,
	)
	if err != nil {
		return models.Policy{}, fmt.Errorf("failed to get policy: %w", err)
	}
	copy(p.ID[:], fetchedID[:])
	copy(p.ContactPointID[:], contactID[:])
	return p, nil
}

// GetPoliciesByUserID returns all active policies for a given user.
func (d *DB) GetPoliciesByUserID(ctx context.Context, userID int64) ([]models.Policy, error) {
	query := `
	SELECT np.id, np.contact_point_id, np.severity, np.status, np.action,
	       np.created_at, np.updated_at, np.condition_type
	FROM notification_policy np
	JOIN contact_points cp ON np.contact_point_id = cp.id
	WHERE cp.user_id = $1 AND np.status = 'active'`

	rows, err := d.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		var fetchedID, contactID uuid.UUID
		err := rows.Scan(
			&fetchedID,
			&contactID,
			&p.Severity,
			&p.Status,
			&p.Action,
			&p.CreatedAt,
			&p.UpdatedAt,
			&p.ConditionType,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}
		copy(p.ID[:], fetchedID[:])
		copy(p.ContactPointID[:], contactID[:])
		policies = append(policies, p)
	}
	return policies, nil
}

// DeletePolicy marks a policy inactive (soft delete) by its UUID string.
func (d *DB) DeletePolicy(ctx context.Context, idStr string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid policy ID: %w", err)
	}

	query := `
	UPDATE notification_policy
	SET status = 'inactive', updated_at = NOW()
	WHERE id = $1`
	_, err = d.Conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete policy: %w", err)
	}
	return nil
}

// UpdatePolicy updates an existing active policy.
func (d *DB) UpdatePolicy(ctx context.Context, p models.Policy) error {
	id := uuid.UUID(p.ID)
	if id == uuid.Nil {
		return fmt.Errorf("invalid policy ID")
	}
	contactID := uuid.UUID(p.ContactPointID)

	query := `
	UPDATE notification_policy
	SET contact_point_id = $1,
	    severity = $2,
	    status = $3,
	    action = $4,
	    condition_type = $5,
	    updated_at = NOW()
	WHERE id = $6 AND status = 'active'`

	_, err := d.Conn.Exec(ctx, query,
		contactID,
		p.Severity,
		p.Status,
		p.Action,
		p.ConditionType,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update policy: %w", err)
	}
	return nil
}
