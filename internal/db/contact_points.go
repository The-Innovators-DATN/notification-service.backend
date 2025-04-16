package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
	"notification-service/internal/models"
)

func (d *DB) CreateContactPoint(ctx context.Context, cp models.ContactPoint) error {
	query := `
		INSERT INTO contact_points (id, name, user_id, type, configuration, status, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id) DO UPDATE
		SET name = $2, user_id = $3, type = $4, configuration = $5, status = $6`
	_, err := d.Conn.Exec(ctx, query, cp.ID, cp.Name, cp.UserID, cp.Type, cp.Configuration, cp.Status, cp.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to create contact point: %w", err)
	}
	return nil
}

func (d *DB) GetContactPoint(ctx context.Context, id string) (models.ContactPoint, error) {
	var cp models.ContactPoint
	var uuid pgtype.UUID
	query := `
		SELECT id, name, user_id, type, configuration, status, created_at
		FROM contact_points
		WHERE id::text = $1`
	err := d.Conn.QueryRow(ctx, query, id).Scan(
		&uuid, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt,
	)
	if err != nil {
		return models.ContactPoint{}, fmt.Errorf("failed to get contact point: %w", err)
	}
	cp.ID = uuid.Bytes
	return cp, nil
}

func (d *DB) GetContactPointsByUserID(ctx context.Context, userID int64) ([]models.ContactPoint, error) {
	rows, err := d.Conn.Query(ctx, `
		SELECT id, name, user_id, type, configuration, status, created_at
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
			&uuid, &cp.Name, &cp.UserID, &cp.Type, &cp.Configuration, &cp.Status, &cp.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan contact point: %w", err)
		}
		cp.ID = uuid.Bytes
		contactPoints = append(contactPoints, cp)
	}

	return contactPoints, nil
}
