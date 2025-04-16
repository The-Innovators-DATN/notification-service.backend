package db

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type DB struct {
	Conn *pgx.Conn
}

func New(dsn string) (*DB, error) {
	conn, err := pgx.Connect(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	return &DB{Conn: conn}, nil
}

func (d *DB) Close() error {
	if err := d.Conn.Close(context.Background()); err != nil {
		return fmt.Errorf("failed to close database connection: %w", err)
	}
	return nil
}
