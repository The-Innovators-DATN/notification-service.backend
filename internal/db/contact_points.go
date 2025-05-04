package db

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"notification-service/internal/models"
)

// CreateContactPoint inserts a new contact point or updates an existing one.
func (d *DB) CreateContactPoint(ctx context.Context, cp models.ContactPoint) (models.ContactPoint, error) {
	// Ensure ID is set
	if cp.ID == [16]byte{} {
		newID := uuid.New()
		copy(cp.ID[:], newID[:])
	}

	query := `
	INSERT INTO contact_points (
		id, name, user_id, type, configuration, status, created_at, updated_at
	)
	VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
	RETURNING id, created_at, updated_at`

	var created models.ContactPoint
	err := d.Conn.QueryRow(ctx, query,
		uuid.UUID(cp.ID),
		cp.Name,
		cp.UserID,
		cp.Type,
		cp.Configuration, // Directly bind the map as JSONB
		cp.Status,
	).Scan(&created.ID, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to create contact point: %w", err)
	}

	created.Name = cp.Name
	created.UserID = cp.UserID
	created.Type = cp.Type
	created.Configuration = cp.Configuration
	created.Status = cp.Status

	return created, nil
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
	id := uuid.UUID(cp.ID)
	if id == uuid.Nil {
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
	WHERE id = $6`

	_, err := d.Conn.Exec(ctx, query,
		cp.Name,
		cp.UserID,
		cp.Type,
		cp.Configuration, // Directly bind the map as JSONB
		cp.Status,
		id,
	)
	if err != nil {
		return fmt.Errorf("failed to update contact point: %w", err)
	}
	return nil
}
