package db

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreateContactPoint inserts a new contact point or updates an existing one.
func (d *DB) CreateContactPoint(ctx context.Context, cp models.ContactPoint) (models.ContactPoint, error) {
	// Validate input
	if cp.ID == [16]byte{} {
		newID := uuid.New()
		copy(cp.ID[:], newID[:])
	}
	if cp.Name == "" {
		return models.ContactPoint{}, fmt.Errorf("name cannot be empty")
	}
	if cp.Type == "" {
		return models.ContactPoint{}, fmt.Errorf("type cannot be empty")
	}
	if cp.Status == "" {
		return models.ContactPoint{}, fmt.Errorf("status cannot be empty")
	}

	// Convert to uuid.UUID for DB binding
	idUUID := uuid.UUID(cp.ID)

	query := `
	INSERT INTO contact_points
	    (id, name, user_id, type, configuration, status, created_at, updated_at)
	VALUES
	    ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	ON CONFLICT (id) DO UPDATE
	    SET name = EXCLUDED.name,
	        user_id = EXCLUDED.user_id,
	        type = EXCLUDED.type,
	        configuration = EXCLUDED.configuration,
	        status = EXCLUDED.status,
	        updated_at = NOW()
	RETURNING id, name, user_id, type, configuration, status, created_at, updated_at`

	var result models.ContactPoint
	var returnedID uuid.UUID
	err := d.Conn.QueryRow(ctx, query,
		idUUID,
		cp.Name,
		cp.UserID,
		cp.Type,
		cp.Configuration,
		cp.Status,
	).Scan(
		&returnedID,
		&result.Name,
		&result.UserID,
		&result.Type,
		&result.Configuration,
		&result.Status,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to create contact point: %w", err)
	}
	// Copy back to model's ID field
	copy(result.ID[:], returnedID[:])

	return result, nil
}

// GetContactPointByID retrieves an active contact point by its UUID string.
func (d *DB) GetContactPointByID(ctx context.Context, idStr string) (models.ContactPoint, error) {
	idUUID, err := uuid.Parse(idStr)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("invalid UUID format: %w", err)
	}

	query := `
	SELECT id, name, user_id, type, configuration, status, created_at, updated_at
	FROM contact_points
	WHERE id = $1 AND status = 'active'`

	var cp models.ContactPoint
	var returnedID uuid.UUID
	err = d.Conn.QueryRow(ctx, query, idUUID).Scan(
		&returnedID,
		&cp.Name,
		&cp.UserID,
		&cp.Type,
		&cp.Configuration,
		&cp.Status,
		&cp.CreatedAt,
		&cp.UpdatedAt,
	)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to get contact point: %w", err)
	}
	copy(cp.ID[:], returnedID[:])
	return cp, nil
}

// GetContactPointsByUserID returns all active contact points for a user.
func (d *DB) GetContactPointsByUserID(ctx context.Context, userID int64) ([]models.ContactPoint, error) {
	query := `
	SELECT id, name, user_id, type, configuration, status, created_at, updated_at
	FROM contact_points
	WHERE user_id = $1 AND status = 'active'`

	rows, err := d.Conn.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact points by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var cps []models.ContactPoint
	for rows.Next() {
		var cp models.ContactPoint
		var returnedID uuid.UUID
		err := rows.Scan(
			&returnedID,
			&cp.Name,
			&cp.UserID,
			&cp.Type,
			&cp.Configuration,
			&cp.Status,
			&cp.CreatedAt,
			&cp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact point: %w", err)
		}
		copy(cp.ID[:], returnedID[:])
		cps = append(cps, cp)
	}

	return cps, nil
}

// DeleteContactPoint performs a soft-delete by marking status and updating timestamp.
func (d *DB) DeleteContactPoint(ctx context.Context, idStr string) error {
	idUUID, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("invalid UUID format: %w", err)
	}

	query := `
	UPDATE contact_points
	SET status = 'deleted', updated_at = NOW()
	WHERE id = $1`
	_, err = d.Conn.Exec(ctx, query, idUUID)
	if err != nil {
		return fmt.Errorf("failed to delete contact point: %w", err)
	}
	return nil
}

// UpdateContactPoint updates fields of an existing active contact point.
func (d *DB) UpdateContactPoint(ctx context.Context, cp models.ContactPoint) error {
	// Ensure ID is valid
	idUUID := uuid.UUID(cp.ID)
	if idUUID == uuid.Nil {
		return fmt.Errorf("invalid contact point ID")
	}

	query := `
	UPDATE contact_points
	SET name = $1,
	    user_id = $2,
	    type = $3,
	    configuration = $4,
	    status = $5,
	    updated_at = NOW()
	WHERE id = $6 AND status = 'active'`
	_, err := d.Conn.Exec(ctx, query,
		cp.Name,
		cp.UserID,
		cp.Type,
		cp.Configuration,
		cp.Status,
		idUUID,
	)
	if err != nil {
		return fmt.Errorf("failed to update contact point: %w", err)
	}
	return nil
}
