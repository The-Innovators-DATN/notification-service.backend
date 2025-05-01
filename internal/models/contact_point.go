package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type ContactPoint struct {
	ID            [16]byte  `json:"id"`
	Name          string    `json:"name"`
	UserID        int64     `json:"user_id"`
	Type          string    `json:"type"`
	Configuration string    `json:"configuration"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (cp ContactPoint) MarshalJSON() ([]byte, error) {
	type Alias ContactPoint
	return json.Marshal(&struct {
		ID string `json:"id"`
		*Alias
	}{
		ID:    uuid.UUID(cp.ID).String(),
		Alias: (*Alias)(&cp),
	})
}
