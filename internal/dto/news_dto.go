package dto

import "time"

type CreateNewsRequest struct {
	Title string `json:"title" binding:"required,min=3,max=255"`

	Excerpt string `json:"excerpt" binding:"max=1000"`

	Content string `json:"content" binding:"required,min=10,max=100000"`

	FeaturedImageURL string `json:"featured_image_url" binding:"omitempty,url,max=2000"`

	Status string `json:"status" binding:"required,oneof=draft published archived"`

	IsFeatured bool `json:"is_featured"`
}

type UpdateNewsRequest struct {
	Title string `json:"title" binding:"required,min=3,max=255"`

	Excerpt string `json:"excerpt" binding:"max=1000"`

	Content string `json:"content" binding:"required,min=10,max=100000"`

	FeaturedImageURL string `json:"featured_image_url" binding:"omitempty,url,max=2000"`

	Status string `json:"status" binding:"required,oneof=draft published archived"`

	IsFeatured bool `json:"is_featured"`
}

type NewsResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Excerpt          string     `json:"excerpt,omitempty"`
	Content          string     `json:"content"`
	FeaturedImageURL string     `json:"featured_image_url,omitempty"`
	Status           string     `json:"status"`
	IsFeatured       bool       `json:"is_featured"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type NewsListQuery struct {
	Page       int
	Limit      int
	Search     string
	Status     string
	IsFeatured *bool
}