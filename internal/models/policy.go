package models

import "time"

type Policy struct {
	ID             [16]byte  `json:"id"`
	ContactPointID [16]byte  `json:"contact_point_id"`
	Severity       int16     `json:"severity"`
	Status         string    `json:"status"`
	Action         string    `json:"action"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
	ConditionType  string    `json:"condition_type"`
}
