package models

import (
	"encoding/json"
	"github.com/google/uuid"
	"time"
)

type Policy struct {
	ID             [16]byte      `json:"id"`
	ContactPointID [16]byte      `json:"contact_point_id"`
	Severity       int           `json:"severity"`
	Status         string        `json:"status"`
	Action         string        `json:"action"`
	CreatedAt      time.Time     `json:"created_at"`
	UpdatedAt      time.Time     `json:"updated_at"`
	ConditionType  string        `json:"condition_type"`
	ContactPoint   *ContactPoint `json:"contact_point,omitempty"` // Added for response, not stored in DB
}

func (p Policy) MarshalJSON() ([]byte, error) {
	type Alias Policy
	return json.Marshal(&struct {
		ID             string `json:"id"`
		ContactPointID string `json:"contact_point_id"`
		*Alias
	}{
		ID:             uuid.UUID(p.ID).String(),
		ContactPointID: uuid.UUID(p.ContactPointID).String(),
		Alias:          (*Alias)(&p),
	})
}
