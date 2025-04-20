package db

import (
	"context"
	"database/sql"
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
		id, contact_point_id, severity, status, action, condition_type, created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	ON CONFLICT (id) DO UPDATE
	SET contact_point_id = EXCLUDED.contact_point_id,
	    severity         = EXCLUDED.severity,
	    status           = EXCLUDED.status,
	    action           = EXCLUDED.action,
	    condition_type   = EXCLUDED.condition_type,
	    updated_at       = NOW()`

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

// GetPolicyByID retrieves an active policy and its contact point (if active).
func (d *DB) GetPolicyByID(ctx context.Context, idStr string) (models.Policy, error) {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return models.Policy{}, fmt.Errorf("invalid policy ID: %w", err)
	}

	query := `
	SELECT
		p.id, p.contact_point_id, p.severity, p.status, p.action, p.condition_type, p.created_at, p.updated_at,
		cp.id, cp.name, cp.user_id, cp.type, cp.configuration, cp.status, cp.created_at, cp.updated_at
	FROM notification_policy p
	LEFT JOIN contact_points cp
	  ON p.contact_point_id = cp.id AND cp.status = 'active'
	WHERE p.id = $1 AND p.status = 'active'`

	row := d.Conn.QueryRow(ctx, query, id)

	var p models.Policy
	var cpID sql.NullString
	var cpName, cpType, cpConfig, cpStatus sql.NullString
	var cpUserID sql.NullInt64
	var cpCreated, cpUpdated sql.NullTime

	err = row.Scan(
		&p.ID,
		&p.ContactPointID,
		&p.Severity,
		&p.Status,
		&p.Action,
		&p.ConditionType,
		&p.CreatedAt,
		&p.UpdatedAt,
		&cpID,
		&cpName,
		&cpUserID,
		&cpType,
		&cpConfig,
		&cpStatus,
		&cpCreated,
		&cpUpdated,
	)
	if err != nil {
		return models.Policy{}, fmt.Errorf("failed to get policy: %w", err)
	}

	// Populate nested ContactPoint only if present
	if cpID.Valid {
		uid, _ := uuid.Parse(cpID.String)
		var cp models.ContactPoint
		copy(cp.ID[:], uid[:])
		cp.Name = cpName.String
		cp.UserID = cpUserID.Int64
		cp.Type = cpType.String
		cp.Configuration = cpConfig.String
		cp.Status = cpStatus.String
		cp.CreatedAt = cpCreated.Time
		cp.UpdatedAt = cpUpdated.Time
		p.ContactPoint = &cp
	}

	return p, nil
}

// GetPoliciesByUserID returns all active policies (and their contact points) for a user.
func (d *DB) GetPoliciesByUserID(ctx context.Context, userID int64) ([]models.Policy, error) {
	query := `
	SELECT
		np.id, np.contact_point_id, np.severity, np.status, np.action, np.condition_type, np.created_at, np.updated_at,
		cp.id, cp.name, cp.user_id, cp.type, cp.configuration, cp.status, cp.created_at, cp.updated_at
	FROM notification_policy np
	LEFT JOIN contact_points cp
	  ON np.contact_point_id = cp.id AND cp.user_id = $1 AND cp.status = 'active'
	WHERE np.status = 'active'`

	rows, err := d.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get policies by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var policies []models.Policy
	for rows.Next() {
		var p models.Policy
		var cpID sql.NullString
		var cpName, cpType, cpConfig, cpStatus sql.NullString
		var cpUserID sql.NullInt64
		var cpCreated, cpUpdated sql.NullTime

		err := rows.Scan(
			&p.ID,
			&p.ContactPointID,
			&p.Severity,
			&p.Status,
			&p.Action,
			&p.ConditionType,
			&p.CreatedAt,
			&p.UpdatedAt,
			&cpID,
			&cpName,
			&cpUserID,
			&cpType,
			&cpConfig,
			&cpStatus,
			&cpCreated,
			&cpUpdated,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan policy: %w", err)
		}

		if cpID.Valid {
			uid, _ := uuid.Parse(cpID.String)
			var cp models.ContactPoint
			copy(cp.ID[:], uid[:])
			cp.Name = cpName.String
			cp.UserID = cpUserID.Int64
			cp.Type = cpType.String
			cp.Configuration = cpConfig.String
			cp.Status = cpStatus.String
			cp.CreatedAt = cpCreated.Time
			cp.UpdatedAt = cpUpdated.Time
			p.ContactPoint = &cp
		}

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
