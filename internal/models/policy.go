package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

// Policy represents a services policy with associated contact point.
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

// PolicyCreate represents the input structure for creating a new policy.
type PolicyCreate struct {
	ContactPointID string `json:"contact_point_id" binding:"required"`
	Severity       int    `json:"severity" binding:"required"`
	Action         string `json:"action" binding:"required"`
	ConditionType  string `json:"condition_type" binding:"required"`
}

// PolicyUpdate represents the input structure for updating an existing policy.
type PolicyUpdate struct {
	ID             string `json:"id" binding:"required"`
	ContactPointID string `json:"contact_point_id" binding:"required"`
	Severity       int    `json:"severity,omitempty"`
	Status         string `json:"status,omitempty"`
	Action         string `json:"action,omitempty"`
	ConditionType  string `json:"condition_type,omitempty"`
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

// UnmarshalJSON customizes JSON deserialization for Policy to convert string IDs to [16]byte.
func (p *Policy) UnmarshalJSON(data []byte) error {
	type Alias Policy
	aux := &struct {
		ID             string `json:"id"`
		ContactPointID string `json:"contact_point_id"`
		*Alias
	}{
		Alias: (*Alias)(p),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ID != "" {
		parsedID, err := uuid.Parse(aux.ID)
		if err != nil {
			return fmt.Errorf("invalid UUID format for ID: %w", err)
		}
		copy(p.ID[:], parsedID[:])
	}
	if aux.ContactPointID != "" {
		parsedContactPointID, err := uuid.Parse(aux.ContactPointID)
		if err != nil {
			return fmt.Errorf("invalid UUID format for ContactPointID: %w", err)
		}
		copy(p.ContactPointID[:], parsedContactPointID[:])
	}
	return nil
}
