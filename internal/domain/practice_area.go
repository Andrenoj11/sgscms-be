package domain

import "time"

type PracticeArea struct {
	ID           string
	Name         string
	Slug         string
	Description  string
	IsActive     bool
	DisplayOrder int
	CreatedBy    *string
	UpdatedBy    *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}