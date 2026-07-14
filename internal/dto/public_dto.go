package dto

import "time"

type PublicListQuery struct {
	Page   int
	Limit  int
	Search string
}

type PublicNewsListQuery struct {
	Page       int
	Limit      int
	Search     string
	IsFeatured *bool
}

type PublicPracticeAreaResponse struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Slug         string `json:"slug"`
	Description  string `json:"description,omitempty"`
	DisplayOrder int    `json:"display_order"`
}

type PublicTeamListResponse struct {
	ID               string                       `json:"id"`
	Name             string                       `json:"name"`
	Slug             string                       `json:"slug"`
	Degree           string                       `json:"degree,omitempty"`
	Position         string                       `json:"position"`
	ShortDescription string                       `json:"short_description,omitempty"`
	PhotoURL         string                       `json:"photo_url,omitempty"`
	DisplayOrder     int                          `json:"display_order"`
	PracticeAreas    []PublicPracticeAreaResponse `json:"practice_areas"`
}

type PublicTeamDetailResponse struct {
	ID               string                       `json:"id"`
	Name             string                       `json:"name"`
	Slug             string                       `json:"slug"`
	Degree           string                       `json:"degree,omitempty"`
	Position         string                       `json:"position"`
	ShortDescription string                       `json:"short_description,omitempty"`
	Biography        string                       `json:"biography,omitempty"`
	PhotoURL         string                       `json:"photo_url,omitempty"`
	Email            string                       `json:"email,omitempty"`
	LinkedInURL      string                       `json:"linkedin_url,omitempty"`
	DisplayOrder     int                          `json:"display_order"`
	PublishedAt      *time.Time                   `json:"published_at,omitempty"`
	PracticeAreas    []PublicPracticeAreaResponse `json:"practice_areas"`
}

type PublicNewsListResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Excerpt          string     `json:"excerpt,omitempty"`
	FeaturedImageURL string     `json:"featured_image_url,omitempty"`
	IsFeatured       bool       `json:"is_featured"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
}

type PublicNewsDetailResponse struct {
	ID               string     `json:"id"`
	Title            string     `json:"title"`
	Slug             string     `json:"slug"`
	Excerpt          string     `json:"excerpt,omitempty"`
	Content          string     `json:"content"`
	FeaturedImageURL string     `json:"featured_image_url,omitempty"`
	IsFeatured       bool       `json:"is_featured"`
	PublishedAt      *time.Time `json:"published_at,omitempty"`
}