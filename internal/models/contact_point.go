package models

import (
	"encoding/json"
	"fmt"
	"github.com/google/uuid"
	"time"
)

type ContactPoint struct {
	ID            [16]byte               `json:"id"`
	Name          string                 `json:"name"`
	UserID        int                    `json:"user_id"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration"` // Stored as string in DB
	Status        string                 `json:"status"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

type ContactPointCreate struct {
	Name          string                 `json:"name" binding:"required"`
	UserID        int                    `json:"user_id" binding:"required"`
	Type          string                 `json:"type" binding:"required"`
	Configuration map[string]interface{} `json:"configuration" binding:"required"`
}

type ContactPointUpdate struct {
	ID            string                 `json:"id" binding:"required"`
	Name          string                 `json:"name,omitempty"`
	UserID        *int                   `json:"user_id,omitempty"`
	Type          string                 `json:"type,omitempty"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Status        string                 `json:"status,omitempty"`
}

func (cp ContactPoint) MarshalJSON() ([]byte, error) {
	type Alias ContactPoint
	return json.Marshal(&struct {
		ID            string                 `json:"id"`
		Configuration map[string]interface{} `json:"configuration"`
		*Alias
	}{
		ID:            uuid.UUID(cp.ID).String(),
		Configuration: cp.Configuration,
		Alias:         (*Alias)(&cp),
	})
}

func (cp *ContactPoint) UnmarshalJSON(data []byte) error {
	type Alias ContactPoint
	aux := &struct {
		ID            string                 `json:"id"`
		Configuration map[string]interface{} `json:"configuration"`
		*Alias
	}{
		Alias: (*Alias)(cp),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if aux.ID != "" {
		parsedID, err := uuid.Parse(aux.ID)
		if err != nil {
			return fmt.Errorf("invalid UUID format for ID: %w", err)
		}
		copy(cp.ID[:], parsedID[:])
	}
	cp.Configuration = aux.Configuration // Directly assign the parsed configuration
	return nil
}
