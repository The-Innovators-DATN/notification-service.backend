package models

import "time"

type ContactPoint struct {
	ID            [16]byte  `json:"id"`
	Name          string    `json:"name"`
	UserID        int64     `json:"user_id"`
	Type          string    `json:"type"`
	Configuration string    `json:"configuration"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
