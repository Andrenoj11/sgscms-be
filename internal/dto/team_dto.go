package dto

import "time"

type CreateTeamRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`

	Degree string `json:"degree" binding:"max=100"`

	Position string `json:"position" binding:"required,min=2,max=150"`

	ShortDescription string `json:"short_description" binding:"max=1000"`

	Biography string `json:"biography" binding:"max=20000"`

	PhotoURL string `json:"photo_url" binding:"omitempty,url,max=2000"`

	Email string `json:"email" binding:"omitempty,email,max=150"`

	LinkedInURL string `json:"linkedin_url" binding:"omitempty,url,max=2000"`

	DisplayOrder int `json:"display_order" binding:"min=0"`

	IsPublished bool `json:"is_published"`

	PracticeAreaIDs []string `json:"practice_area_ids"`
}

type UpdateTeamRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`

	Degree string `json:"degree" binding:"max=100"`

	Position string `json:"position" binding:"required,min=2,max=150"`

	ShortDescription string `json:"short_description" binding:"max=1000"`

	Biography string `json:"biography" binding:"max=20000"`

	PhotoURL string `json:"photo_url" binding:"omitempty,url,max=2000"`

	Email string `json:"email" binding:"omitempty,email,max=150"`

	LinkedInURL string `json:"linkedin_url" binding:"omitempty,url,max=2000"`

	DisplayOrder int `json:"display_order" binding:"min=0"`

	IsPublished bool `json:"is_published"`

	PracticeAreaIDs []string `json:"practice_area_ids"`
}

type TeamResponse struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Slug             string                 `json:"slug"`
	Degree           string                 `json:"degree,omitempty"`
	Position         string                 `json:"position"`
	ShortDescription string                 `json:"short_description,omitempty"`
	Biography        string                 `json:"biography,omitempty"`
	PhotoURL         string                 `json:"photo_url,omitempty"`
	Email            string                 `json:"email,omitempty"`
	LinkedInURL      string                 `json:"linkedin_url,omitempty"`
	DisplayOrder     int                    `json:"display_order"`
	IsPublished      bool                   `json:"is_published"`
	PublishedAt      *time.Time             `json:"published_at,omitempty"`
	PracticeAreas    []PracticeAreaResponse `json:"practice_areas"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type TeamListQuery struct {
	Page      int
	Limit     int
	Search    string
	Published *bool
}