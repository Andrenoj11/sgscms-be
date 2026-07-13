package domain

import (
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrInvalid      = errors.New("invalid input")
)

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	Name         string     `json:"name"`
	PasswordHash string     `json:"-"`
	Active       bool       `json:"isActive"`
	Permissions  []string   `json:"permissions,omitempty"`
	LastLoginAt  *time.Time `json:"lastLoginAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
}
type Session struct {
	ID                  uuid.UUID
	UserID              uuid.UUID
	TokenHash, CSRFHash string
	ExpiresAt           time.Time
}
type Category struct {
	ID           uuid.UUID `json:"id"`
	Name         string    `json:"name"`
	Slug         string    `json:"slug"`
	Type         string    `json:"type"`
	Description  string    `json:"description"`
	DisplayOrder int       `json:"displayOrder"`
	Active       bool      `json:"isActive"`
}
type Tag struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
	Slug string    `json:"slug"`
}
type Media struct {
	ID           uuid.UUID `json:"id"`
	FileName     string    `json:"fileName"`
	OriginalName string    `json:"originalName"`
	MIMEType     string    `json:"mimeType"`
	URL          string    `json:"url"`
	AltText      string    `json:"altText"`
	Caption      string    `json:"caption"`
	FileSize     int64     `json:"fileSize"`
	Width        int       `json:"width"`
	Height       int       `json:"height"`
	CreatedBy    uuid.UUID `json:"createdBy"`
	CreatedAt    time.Time `json:"createdAt"`
}

type NewsStatus string

const (
	NewsDraft     NewsStatus = "draft"
	NewsReview    NewsStatus = "in_review"
	NewsScheduled NewsStatus = "scheduled"
	NewsPublished NewsStatus = "published"
	NewsArchived  NewsStatus = "archived"
)

type News struct {
	ID              uuid.UUID  `json:"id"`
	Title           string     `json:"title"`
	Slug            string     `json:"slug"`
	Excerpt         string     `json:"excerpt"`
	Content         string     `json:"content"`
	FeaturedImageID *uuid.UUID `json:"featuredImageId,omitempty"`
	CategoryID      *uuid.UUID `json:"categoryId,omitempty"`
	Category        *Category  `json:"category,omitempty"`
	Tags            []Tag      `json:"tags,omitempty"`
	Status          NewsStatus `json:"status"`
	Featured        bool       `json:"isFeatured"`
	PublishedAt     *time.Time `json:"publishedAt,omitempty"`
	ScheduledAt     *time.Time `json:"scheduledAt,omitempty"`
	CreatedBy       uuid.UUID  `json:"createdBy"`
	AuthorName      string     `json:"authorName"`
	MetaTitle       string     `json:"metaTitle"`
	MetaDescription string     `json:"metaDescription"`
	CanonicalURL    string     `json:"canonicalUrl"`
	LegacyID        string     `json:"legacyId"`
	LegacyPath      string     `json:"legacyPath"`
	ViewCount       int64      `json:"viewCount"`
	Version         int        `json:"version"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"-"`
}

func (n News) ValidateForPublication(now time.Time) error {
	if strings.TrimSpace(n.Title) == "" || strings.TrimSpace(n.Slug) == "" || strings.TrimSpace(n.Content) == "" || n.CategoryID == nil || n.CreatedBy == uuid.Nil {
		return ErrInvalid
	}
	if n.Status == NewsScheduled && (n.ScheduledAt == nil || !n.ScheduledAt.After(now)) {
		return ErrInvalid
	}
	return nil
}

type TeamStatus string

const (
	TeamDraft    TeamStatus = "draft"
	TeamActive   TeamStatus = "active"
	TeamInactive TeamStatus = "inactive"
	TeamArchived TeamStatus = "archived"
)

type TeamMember struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Slug            string     `json:"slug"`
	AcademicTitles  string     `json:"academicTitles"`
	Position        string     `json:"position"`
	ShortBio        string     `json:"shortBio"`
	Bio             string     `json:"bio"`
	Email           string     `json:"email"`
	LinkedInURL     string     `json:"linkedInUrl"`
	PhotoID         *uuid.UUID `json:"photoId,omitempty"`
	PracticeAreas   []Category `json:"practiceAreas,omitempty"`
	DisplayOrder    int        `json:"displayOrder"`
	Featured        bool       `json:"isFeatured"`
	Status          TeamStatus `json:"status"`
	MetaTitle       string     `json:"metaTitle"`
	MetaDescription string     `json:"metaDescription"`
	LegacyID        string     `json:"legacyId"`
	LegacyPath      string     `json:"legacyPath"`
	Version         int        `json:"version"`
	CreatedBy       uuid.UUID  `json:"createdBy"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
	DeletedAt       *time.Time `json:"-"`
}

func (t TeamMember) ValidateForActivation() error {
	if strings.TrimSpace(t.Name) == "" || strings.TrimSpace(t.Slug) == "" || strings.TrimSpace(t.Position) == "" || strings.TrimSpace(t.Bio) == "" {
		return ErrInvalid
	}
	return nil
}

type ListOptions struct {
	Page, Limit                   int
	Search, Category, Tag, Status string
	Featured                      *bool
	Sort                          string
}

func (o ListOptions) Normalize() ListOptions {
	if o.Page < 1 {
		o.Page = 1
	}
	if o.Limit < 1 {
		o.Limit = 10
	}
	if o.Limit > 100 {
		o.Limit = 100
	}
	return o
}

type Page[T any] struct {
	Items []T
	Total int64
}
type AuditEvent struct {
	ActorID                                                           *uuid.UUID
	ActorType, Action, EntityType, EntityID, RequestID, IP, UserAgent string
	Before, After                                                     any
}
type Role struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}
type Redirect struct {
	ID              uuid.UUID `json:"id"`
	SourcePath      string    `json:"sourcePath"`
	DestinationPath string    `json:"destinationPath"`
	StatusCode      int       `json:"statusCode"`
	Active          bool      `json:"isActive"`
	CreatedBy       uuid.UUID `json:"createdBy"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
type AuditLog struct {
	ID             uuid.UUID  `json:"id"`
	ActorID        *uuid.UUID `json:"actorId,omitempty"`
	ActorType      string     `json:"actorType"`
	Action         string     `json:"action"`
	EntityType     string     `json:"entityType"`
	EntityID       string     `json:"entityId"`
	RequestID      string     `json:"requestId"`
	IP             string     `json:"ipAddress"`
	UserAgent      string     `json:"userAgent"`
	PreviousValues any        `json:"previousValues,omitempty"`
	NewValues      any        `json:"newValues,omitempty"`
	CreatedAt      time.Time  `json:"createdAt"`
}
type APIClient struct {
	ID          uuid.UUID  `json:"id"`
	ClientID    string     `json:"clientId"`
	KeyID       string     `json:"keyId"`
	Status      string     `json:"status"`
	Permissions []string   `json:"permissions"`
	ExpiresAt   *time.Time `json:"expiresAt,omitempty"`
	LastUsedAt  *time.Time `json:"lastUsedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}
