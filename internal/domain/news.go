package domain

import "time"

type NewsStatus string

const (
	NewsStatusDraft     NewsStatus = "draft"
	NewsStatusPublished NewsStatus = "published"
	NewsStatusArchived  NewsStatus = "archived"
)

type News struct {
	ID               string
	Title            string
	Slug             string
	Excerpt          string
	Content          string
	FeaturedImageURL string
	Status           NewsStatus
	IsFeatured       bool
	PublishedAt      *time.Time
	CreatedBy        *string
	UpdatedBy        *string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	DeletedAt        *time.Time
}