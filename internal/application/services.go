package application

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	_ "golang.org/x/image/webp"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/microcosm-cc/bluemonday"
	"github.com/sgs-law-firm/cms/internal/domain"
)

type AuthService struct {
	Store domain.Store
	TTL   time.Duration
}
type LoginResult struct {
	User        domain.User
	Token, CSRF string
	ExpiresAt   time.Time
}

func (a AuthService) Login(c context.Context, email, password string) (LoginResult, error) {
	u, e := a.Store.UserByEmail(c, strings.TrimSpace(strings.ToLower(email)))
	if e != nil || !u.Active || !VerifyPassword(u.PasswordHash, password) {
		return LoginResult{}, domain.ErrUnauthorized
	}
	token, e := RandomToken(32)
	if e != nil {
		return LoginResult{}, e
	}
	csrf, e := RandomToken(32)
	if e != nil {
		return LoginResult{}, e
	}
	x := time.Now().UTC().Add(a.TTL)
	e = a.Store.CreateSession(c, domain.Session{ID: uuid.New(), UserID: u.ID, TokenHash: SHA256Hex(token), CSRFHash: SHA256Hex(csrf), ExpiresAt: x})
	if e != nil {
		return LoginResult{}, e
	}
	_ = a.Store.TouchLogin(c, u.ID)
	return LoginResult{u, token, csrf, x}, nil
}
func (a AuthService) Logout(c context.Context, token string) error {
	if token == "" {
		return nil
	}
	return a.Store.DeleteSession(c, SHA256Hex(token))
}

type ContentService struct {
	Store     domain.Store
	sanitizer *bluemonday.Policy
}

func NewContentService(s domain.Store) *ContentService {
	return &ContentService{Store: s, sanitizer: bluemonday.UGCPolicy()}
}

var slugBad = regexp.MustCompile(`[^a-z0-9]+`)

func Slug(s string) string {
	return strings.Trim(slugBad.ReplaceAllString(strings.ToLower(strings.TrimSpace(s)), "-"), "-")
}
func (c *ContentService) PrepareNews(n *domain.News) {
	if n.Slug == "" {
		n.Slug = Slug(n.Title)
	} else {
		n.Slug = Slug(n.Slug)
	}
	n.Content = c.sanitizer.Sanitize(n.Content)
	n.Excerpt = c.sanitizer.Sanitize(n.Excerpt)
}
func (c *ContentService) CreateNews(ctx context.Context, n *domain.News, tags []uuid.UUID) error {
	c.PrepareNews(n)
	if n.Status == domain.NewsPublished || n.Status == domain.NewsScheduled {
		if e := n.ValidateForPublication(time.Now().UTC()); e != nil {
			return e
		}
		if n.Status == domain.NewsPublished && n.PublishedAt == nil {
			v := time.Now().UTC()
			n.PublishedAt = &v
		}
	}
	return c.Store.CreateNews(ctx, n, tags)
}
func (c *ContentService) UpdateNews(ctx context.Context, n *domain.News, tags []uuid.UUID) error {
	c.PrepareNews(n)
	if n.Status == domain.NewsPublished || n.Status == domain.NewsScheduled {
		if e := n.ValidateForPublication(time.Now().UTC()); e != nil {
			return e
		}
		if n.Status == domain.NewsPublished && n.PublishedAt == nil {
			v := time.Now().UTC()
			n.PublishedAt = &v
		}
	}
	return c.Store.UpdateNews(ctx, n, tags)
}
func (c *ContentService) CreateTeam(ctx context.Context, t *domain.TeamMember, areas []uuid.UUID) error {
	if t.Slug == "" {
		t.Slug = Slug(t.Name)
	} else {
		t.Slug = Slug(t.Slug)
	}
	t.Bio = c.sanitizer.Sanitize(t.Bio)
	t.ShortBio = c.sanitizer.Sanitize(t.ShortBio)
	if t.DisplayOrder < 1 {
		t.DisplayOrder = 1
	}
	if t.Status == domain.TeamActive {
		if e := t.ValidateForActivation(); e != nil {
			return e
		}
	}
	return c.Store.CreateTeam(ctx, t, areas)
}
func (c *ContentService) UpdateTeam(ctx context.Context, t *domain.TeamMember, areas []uuid.UUID) error {
	if t.Slug == "" {
		t.Slug = Slug(t.Name)
	} else {
		t.Slug = Slug(t.Slug)
	}
	t.Bio = c.sanitizer.Sanitize(t.Bio)
	t.ShortBio = c.sanitizer.Sanitize(t.ShortBio)
	if t.Status == domain.TeamActive {
		if e := t.ValidateForActivation(); e != nil {
			return e
		}
	}
	return c.Store.UpdateTeam(ctx, t, areas)
}

type MediaService struct {
	Store               domain.Store
	Storage             domain.ObjectStorage
	MaxBytes            int64
	MaxWidth, MaxHeight int
}

func (m MediaService) Upload(c context.Context, r io.Reader, size int64, original, alt, caption string, user uuid.UUID) (domain.Media, error) {
	if size <= 0 || size > m.MaxBytes {
		return domain.Media{}, fmt.Errorf("%w: file size", domain.ErrInvalid)
	}
	b, e := io.ReadAll(io.LimitReader(r, m.MaxBytes+1))
	if e != nil || int64(len(b)) > m.MaxBytes {
		return domain.Media{}, domain.ErrInvalid
	}
	kind := http.DetectContentType(b)
	if kind != "image/jpeg" && kind != "image/png" && kind != "image/webp" {
		return domain.Media{}, fmt.Errorf("%w: unsupported image type", domain.ErrInvalid)
	}
	cfg, _, e := image.DecodeConfig(bytes.NewReader(b))
	if e != nil {
		return domain.Media{}, fmt.Errorf("%w: invalid image", domain.ErrInvalid)
	}
	if cfg.Width < 1 || cfg.Height < 1 || cfg.Width > m.MaxWidth || cfg.Height > m.MaxHeight {
		return domain.Media{}, fmt.Errorf("%w: image dimensions", domain.ErrInvalid)
	}
	id := uuid.New()
	ext, _ := mime.ExtensionsByType(kind)
	suffix := ".img"
	if len(ext) > 0 {
		suffix = ext[0]
	}
	key := time.Now().UTC().Format("2006/01/") + id.String() + suffix
	url, e := m.Storage.Put(c, key, bytes.NewReader(b), int64(len(b)), kind)
	if e != nil {
		return domain.Media{}, e
	}
	v := domain.Media{ID: id, FileName: key, OriginalName: filepath.Base(original), MIMEType: kind, FileSize: int64(len(b)), URL: url, AltText: alt, Caption: caption, Width: cfg.Width, Height: cfg.Height, CreatedBy: user, CreatedAt: time.Now().UTC()}
	if e = m.Store.CreateMedia(c, &v); e != nil {
		_ = m.Storage.Delete(c, key)
		return domain.Media{}, e
	}
	return v, nil
}

type SignatureInput struct{ Version, KeyID, Timestamp, Nonce, Method, URI, ContentType, BodyHash, Signature string }
type SignatureVerifier struct {
	Store     domain.Store
	Tolerance time.Duration
}

func (v SignatureVerifier) Verify(c context.Context, in SignatureInput, required string) ([]string, error) {
	if in.Version != "v1" || in.KeyID == "" || in.Nonce == "" {
		return nil, domain.ErrUnauthorized
	}
	ts, e := time.Parse(time.RFC3339, in.Timestamp)
	if e != nil || time.Since(ts) > v.Tolerance || time.Until(ts) > v.Tolerance {
		return nil, domain.ErrUnauthorized
	}
	_, secret, perms, e := v.Store.APIClientSecret(c, in.KeyID)
	if e != nil {
		return nil, domain.ErrUnauthorized
	}
	canonical := strings.Join([]string{in.Version, in.KeyID, in.Timestamp, in.Nonce, strings.ToUpper(in.Method), in.URI, in.ContentType, in.BodyHash}, "\n")
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))
	provided, e := hex.DecodeString(in.Signature)
	if e != nil || !hmac.Equal([]byte(expected), []byte(hex.EncodeToString(provided))) {
		return nil, domain.ErrUnauthorized
	}
	if required != "" && !HasPermission(perms, required) {
		return nil, domain.ErrForbidden
	}
	if e = v.Store.RegisterNonce(c, in.KeyID, in.Nonce, ts.Add(v.Tolerance)); e != nil {
		return nil, domain.ErrUnauthorized
	}
	return perms, nil
}
func HasPermission(perms []string, required string) bool {
	for _, p := range perms {
		if p == required || p == "*" {
			return true
		}
	}
	return false
}
