package models

import "time"

type Policy struct {
	ID             [16]byte  `json:"id"`
	ContactPointID [16]byte  `json:"contact_point_id"`
	Severity       int16     `json:"severity"`
	Status         string    `json:"status"`
	Topic          string    `json:"topic"`
	CreatedAt      time.Time `json:"created_at"`
}
