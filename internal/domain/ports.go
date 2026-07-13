package domain

import (
	"context"
	"github.com/google/uuid"
	"io"
	"time"
)

type Store interface {
	Ping(context.Context) error
	UserByEmail(context.Context, string) (User, error)
	UserBySessionToken(context.Context, string) (User, error)
	CreateSession(context.Context, Session) error
	DeleteSession(context.Context, string) error
	TouchLogin(context.Context, uuid.UUID) error
	PublicNews(context.Context, ListOptions) (Page[News], error)
	PublicNewsBySlug(context.Context, string) (News, error)
	AdminNews(context.Context, ListOptions) (Page[News], error)
	NewsByID(context.Context, uuid.UUID) (News, error)
	CreateNews(context.Context, *News, []uuid.UUID) error
	UpdateNews(context.Context, *News, []uuid.UUID) error
	SoftDeleteNews(context.Context, uuid.UUID, int) error
	RestoreNews(context.Context, uuid.UUID, int) error
	PublishDueNews(context.Context, time.Time) (int64, error)
	PublicTeam(context.Context, ListOptions) (Page[TeamMember], error)
	PublicTeamBySlug(context.Context, string) (TeamMember, error)
	AdminTeam(context.Context, ListOptions) (Page[TeamMember], error)
	TeamByID(context.Context, uuid.UUID) (TeamMember, error)
	CreateTeam(context.Context, *TeamMember, []uuid.UUID) error
	UpdateTeam(context.Context, *TeamMember, []uuid.UUID) error
	SoftDeleteTeam(context.Context, uuid.UUID, int) error
	RestoreTeam(context.Context, uuid.UUID, int) error
	Categories(context.Context, string, bool) ([]Category, error)
	CreateCategory(context.Context, *Category) error
	Tags(context.Context) ([]Tag, error)
	CreateTag(context.Context, *Tag) error
	CreateMedia(context.Context, *Media) error
	MediaReferencedByPublished(context.Context, uuid.UUID) (bool, error)
	RegisterNonce(context.Context, string, string, time.Time) error
	APIClientSecret(context.Context, string) (keyID string, secret []byte, permissions []string, err error)
	CreateAudit(context.Context, AuditEvent) error
	Users(context.Context, ListOptions) (Page[User], error)
	CreateUser(context.Context, *User) error
	SetUserActive(context.Context, uuid.UUID, bool) error
	AssignUserRoles(context.Context, uuid.UUID, []uuid.UUID) error
	Roles(context.Context) ([]Role, error)
	CreateRole(context.Context, *Role) error
	AuditLogs(context.Context, ListOptions) (Page[AuditLog], error)
	Redirects(context.Context) ([]Redirect, error)
	CreateRedirect(context.Context, *Redirect) error
	UpdateRedirect(context.Context, *Redirect) error
	RedirectWouldLoop(context.Context, string, string, *uuid.UUID) (bool, error)
	APIClients(context.Context) ([]APIClient, error)
	CreateAPIClient(context.Context, *APIClient, string) error
	RevokeAPIClient(context.Context, uuid.UUID) error
}
type ObjectStorage interface {
	Put(context.Context, string, io.Reader, int64, string) (string, error)
	Delete(context.Context, string) error
}
