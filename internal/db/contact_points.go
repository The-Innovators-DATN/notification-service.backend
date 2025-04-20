package db

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgtype"
	"notification-service/internal/models"
)

func (d *DB) CreateContactPoint(ctx context.Context, cp models.ContactPoint) (models.ContactPoint, error) {
	// Validate input
	if cp.ID == [16]byte{} {
		return models.ContactPoint{}, fmt.Errorf("contact point ID cannot be empty")
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

	query := `
        INSERT INTO contact_points (id, name, user_id, type, configuration, status, created_at, updated_at)
        VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
        ON CONFLICT (id) DO UPDATE
        SET name = $2,
            user_id = $3,
            type = $4,
            configuration = $5,
            status = $6,
            updated_at = NOW()
        RETURNING id, name, user_id, type, configuration, status, created_at, updated_at`

	var result models.ContactPoint
	err := d.Conn.QueryRow(ctx, query, cp.ID, cp.Name, cp.UserID, cp.Type, cp.Configuration, cp.Status).
		Scan(&result.ID, &result.Name, &result.UserID, &result.Type, &result.Configuration, &result.Status, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to create contact point: %w", err)
	}

	return result, nil
}

func (d *DB) GetContactPointById(ctx context.Context, id string) (models.ContactPoint, error) {
	var cp models.ContactPoint
	var uuid pgtype.UUID
	query := `
		SELECT id, name, user_id, type, configuration, status, created_at, updated_at
		FROM contact_points
		WHERE id::text = $1 AND status = 'active'`
	err := d.Conn.QueryRow(ctx, query, id).Scan(
		&uuid, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt, &cp.UpdatedAt,
	)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to get contact point: %w", err)
	}
	cp.ID = uuid.Bytes
	return cp, nil
}

func (d *DB) GetContactPointsByUserID(ctx context.Context, userID int64) ([]models.ContactPoint, error) {
	rows, err := d.Conn.Query(ctx, `
		SELECT id, name, user_id, type, configuration, status, created_at, updated_at
		FROM contact_points
		WHERE user_id = $1 AND status = 'active'`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get contact points by user_id %d: %w", userID, err)
	}
	defer rows.Close()

	var contactPoints []models.ContactPoint
	for rows.Next() {
		var cp models.ContactPoint
		var uuid pgtype.UUID
		err := rows.Scan(
			&uuid, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt, &cp.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact point: %w", err)
		}
		cp.ID = uuid.Bytes
		contactPoints = append(contactPoints, cp)
	}

	return contactPoints, nil
}

func (d *DB) DeleteContactPoint(ctx context.Context, id string) error {
	query := `
		UPDATE contact_points
		SET status = 'deleted'
		WHERE id::text = $1`
	_, err := d.Conn.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete contact point: %w", err)
	}
	return nil
}

func (d *DB) UpdateContactPoint(ctx context.Context, cp models.ContactPoint) error {
	query := `
		UPDATE contact_points
		SET name = $1, user_id = $2, type = $3, configuration = $4, status = $5, updated_at = NOW()
		WHERE id::text = $6 AND status = 'active'`
	_, err := d.Conn.Exec(ctx, query, cp.Name, cp.UserID, cp.Type, cp.Configuration, cp.Status, cp.ID)
	if err != nil {
		return fmt.Errorf("failed to update contact point: %w", err)
	}
	return nil
}
