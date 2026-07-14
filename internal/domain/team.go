package domain

import "time"

type Team struct {
	ID               string
	Name             string
	Slug             string
	Degree           string
	Position         string
	ShortDescription string
	Biography        string
	PhotoURL         string
	Email            string
	LinkedInURL      string
	DisplayOrder     int
	IsPublished      bool
	PublishedAt      *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
	PracticeAreas    []PracticeArea
}