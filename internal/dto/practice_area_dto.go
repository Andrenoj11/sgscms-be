package dto

import "time"

type CreatePracticeAreaRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`

	Description string `json:"description" binding:"max=2000"`

	IsActive *bool `json:"is_active"`

	DisplayOrder int `json:"display_order" binding:"min=0"`
}

type UpdatePracticeAreaRequest struct {
	Name string `json:"name" binding:"required,min=2,max=150"`

	Description string `json:"description" binding:"max=2000"`

	IsActive bool `json:"is_active"`

	DisplayOrder int `json:"display_order" binding:"min=0"`
}

type PracticeAreaResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Description  string    `json:"description,omitempty"`
	IsActive     bool      `json:"is_active"`
	DisplayOrder int       `json:"display_order"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type PracticeAreaListQuery struct {
	Page   int
	Limit  int
	Search string
	Active *bool
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}