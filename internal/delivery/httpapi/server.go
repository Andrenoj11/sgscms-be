package httpapi

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"
	"github.com/sgs-law-firm/cms/internal/application"
	"github.com/sgs-law-firm/cms/internal/config"
	"github.com/sgs-law-firm/cms/internal/domain"
)

type Server struct {
	cfg     config.Config
	store   domain.Store
	auth    application.AuthService
	content *application.ContentService
	media   application.MediaService
	signer  application.SignatureVerifier
	log     *slog.Logger
	limiter *ipLimiter
}

func New(cfg config.Config, store domain.Store, storage domain.ObjectStorage, log *slog.Logger) http.Handler {
	s := &Server{cfg: cfg, store: store, auth: application.AuthService{Store: store, TTL: cfg.SessionTTL}, content: application.NewContentService(store), media: application.MediaService{Store: store, Storage: storage, MaxBytes: cfg.MaxUploadBytes, MaxWidth: cfg.MaxImageWidth, MaxHeight: cfg.MaxImageHeight}, signer: application.SignatureVerifier{Store: store, Tolerance: cfg.SignatureTolerance}, log: log, limiter: newIPLimiter(20, time.Minute)}
	r := chi.NewRouter()
	r.Use(middleware.RequestID, middleware.RealIP, middleware.Recoverer, s.security, s.accessLog)
	r.Get("/health/live", s.live)
	r.Get("/health/ready", s.ready)
	r.Handle("/media/*", http.StripPrefix("/media/", http.FileServer(http.Dir(cfg.MediaDir))))
	r.Route("/api/v1/public", func(r chi.Router) {
		r.Use(s.publicCache)
		r.Get("/news", s.publicNews)
		r.Get("/news/featured", s.publicFeaturedNews)
		r.Get("/news/categories", s.publicNewsCategories)
		r.Get("/news/{slug}", s.publicNewsOne)
		r.Get("/team", s.publicTeam)
		r.Get("/team/{slug}", s.publicTeamOne)
		r.Get("/practice-areas", s.publicPracticeAreas)
		r.Get("/practice-areas/{slug}/team", s.publicPracticeTeam)
	})
	r.Route("/api/v1/admin", func(r chi.Router) {
		r.With(s.loginRateLimit).Post("/auth/login", s.login)
		r.Group(func(r chi.Router) {
			r.Use(s.authenticate, s.csrf)
			r.Get("/auth/me", s.me)
			r.Post("/auth/logout", s.logout)
			r.With(s.require("news.read")).Get("/news", s.adminNews)
			r.With(s.require("news.read")).Get("/news/{id}", s.adminNewsOne)
			r.With(s.require("news.create")).Post("/news", s.createNews)
			r.With(s.require("news.update")).Put("/news/{id}", s.updateNews)
			r.With(s.require("news.delete")).Delete("/news/{id}", s.deleteNews)
			r.With(s.require("news.delete")).Post("/news/{id}/restore", s.restoreNews)
			r.With(s.require("team.read")).Get("/team", s.adminTeam)
			r.With(s.require("team.read")).Get("/team/{id}", s.adminTeamOne)
			r.With(s.require("team.create")).Post("/team", s.createTeam)
			r.With(s.require("team.update")).Put("/team/{id}", s.updateTeam)
			r.With(s.require("team.delete")).Delete("/team/{id}", s.deleteTeam)
			r.With(s.require("team.delete")).Post("/team/{id}/restore", s.restoreTeam)
			r.With(s.require("news.read")).Get("/categories", s.adminCategories)
			r.With(s.require("roles.manage")).Post("/categories", s.createCategory)
			r.With(s.require("news.read")).Get("/tags", s.adminTags)
			r.With(s.require("news.create")).Post("/tags", s.createTag)
			r.With(s.require("media.upload")).Post("/media", s.uploadMedia)
			r.With(s.require("users.manage")).Get("/users", s.adminUsers)
			r.With(s.require("users.manage")).Post("/users", s.createUser)
			r.With(s.require("users.manage")).Patch("/users/{id}/active", s.setUserActive)
			r.With(s.require("users.manage")).Put("/users/{id}/roles", s.assignUserRoles)
			r.With(s.require("roles.manage")).Get("/roles", s.adminRoles)
			r.With(s.require("roles.manage")).Post("/roles", s.createRole)
			r.With(s.require("audit.read")).Get("/audit-logs", s.adminAuditLogs)
			r.With(s.require("redirects.manage")).Get("/redirects", s.adminRedirects)
			r.With(s.require("redirects.manage")).Post("/redirects", s.createRedirect)
			r.With(s.require("redirects.manage")).Put("/redirects/{id}", s.updateRedirect)
			r.With(s.require("roles.manage")).Get("/api-clients", s.adminAPIClients)
			r.With(s.require("roles.manage")).Post("/api-clients", s.createAPIClient)
			r.With(s.require("roles.manage")).Post("/api-clients/{id}/revoke", s.revokeAPIClient)
		})
	})
	r.With(s.signed("news.publish")).Post("/api/v1/internal/news/publish-due", s.publishDue)
	return r
}

type ctxKey string

const userKey ctxKey = "user"

func userFrom(c context.Context) domain.User { v, _ := c.Value(userKey).(domain.User); return v }
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, e := r.Cookie(s.cfg.CookieName)
		if e != nil || c.Value == "" {
			fail(w, r, domain.ErrUnauthorized)
			return
		}
		u, e := s.store.UserBySessionToken(r.Context(), application.SHA256Hex(c.Value))
		if e != nil || !u.Active {
			fail(w, r, domain.ErrUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, u)))
	})
}
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		cookie, e := r.Cookie("sgscms_csrf")
		provided := r.Header.Get("X-CSRF-Token")
		if e != nil || provided == "" || !hmac.Equal([]byte(cookie.Value), []byte(provided)) {
			writeError(w, r, http.StatusForbidden, "CSRF_INVALID", "CSRF token is missing or invalid", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}
func (s *Server) require(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !application.HasPermission(userFrom(r.Context()).Permissions, permission) {
				fail(w, r, domain.ErrForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func (s *Server) signed(permission string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, e := io.ReadAll(io.LimitReader(r.Body, 2<<20))
			if e != nil {
				fail(w, r, domain.ErrInvalid)
				return
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			sum := sha256.Sum256(body)
			actual := hex.EncodeToString(sum[:])
			provided := r.Header.Get("X-Body-SHA256")
			if provided == "" {
				provided = actual
			}
			if !hmac.Equal([]byte(actual), []byte(provided)) {
				fail(w, r, domain.ErrUnauthorized)
				return
			}
			_, e = s.signer.Verify(r.Context(), application.SignatureInput{Version: r.Header.Get("X-Signature-Version"), KeyID: r.Header.Get("X-Key-ID"), Timestamp: r.Header.Get("X-Timestamp"), Nonce: r.Header.Get("X-Nonce"), Method: r.Method, URI: r.URL.RequestURI(), ContentType: r.Header.Get("Content-Type"), BodyHash: actual, Signature: r.Header.Get("X-Signature")}, permission)
			if e != nil {
				fail(w, r, e)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func (s *Server) security(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(v int) { w.status = v; w.ResponseWriter.WriteHeader(v) }
func (s *Server) accessLog(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(sw, r)
		s.log.Info("http request", "request_id", middleware.GetReqID(r.Context()), "method", r.Method, "path", r.URL.Path, "status", sw.status, "duration_ms", time.Since(start).Milliseconds())
	})
}
func (s *Server) publicCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=60, stale-while-revalidate=300")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) live(w http.ResponseWriter, r *http.Request) {
	ok(w, r, map[string]string{"status": "ok"})
}
func (s *Server) ready(w http.ResponseWriter, r *http.Request) {
	c, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if s.store.Ping(c) != nil {
		writeError(w, r, 503, "NOT_READY", "Service is not ready", nil)
		return
	}
	ok(w, r, map[string]string{"status": "ready"})
}
func (s *Server) publishDue(w http.ResponseWriter, r *http.Request) {
	n, e := s.store.PublishDueNews(r.Context(), time.Now().UTC())
	result(w, r, map[string]int64{"published": n}, e)
}
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct{ Email, Password string }
	if !decode(w, r, &in) {
		return
	}
	v, e := s.auth.Login(r.Context(), in.Email, in.Password)
	if e != nil {
		_ = s.store.CreateAudit(r.Context(), domain.AuditEvent{ActorType: "anonymous", Action: "login.failed", RequestID: middleware.GetReqID(r.Context()), IP: clientIP(r), UserAgent: r.UserAgent()})
		fail(w, r, e)
		return
	}
	_ = s.store.CreateAudit(r.Context(), domain.AuditEvent{ActorID: &v.User.ID, ActorType: "user", Action: "login.succeeded", EntityType: "user", EntityID: v.User.ID.String(), RequestID: middleware.GetReqID(r.Context()), IP: clientIP(r), UserAgent: r.UserAgent()})
	http.SetCookie(w, &http.Cookie{Name: s.cfg.CookieName, Value: v.Token, Path: "/", Expires: v.ExpiresAt, HttpOnly: true, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteLaxMode})
	http.SetCookie(w, &http.Cookie{Name: "sgscms_csrf", Value: v.CSRF, Path: "/", Expires: v.ExpiresAt, Secure: s.cfg.CookieSecure, SameSite: http.SameSiteStrictMode})
	ok(w, r, map[string]any{"user": v.User, "csrfToken": v.CSRF, "expiresAt": v.ExpiresAt})
}
func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	c, _ := r.Cookie(s.cfg.CookieName)
	if c != nil {
		_ = s.auth.Logout(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: s.cfg.CookieName, Path: "/", MaxAge: -1, HttpOnly: true, Secure: s.cfg.CookieSecure})
	http.SetCookie(w, &http.Cookie{Name: "sgscms_csrf", Path: "/", MaxAge: -1, Secure: s.cfg.CookieSecure})
	ok(w, r, map[string]bool{"loggedOut": true})
}
func (s *Server) me(w http.ResponseWriter, r *http.Request) { ok(w, r, userFrom(r.Context())) }

func options(r *http.Request) domain.ListOptions {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	o := domain.ListOptions{Page: page, Limit: limit, Search: q.Get("search"), Category: q.Get("category"), Tag: q.Get("tag"), Status: q.Get("status")}
	if v := q.Get("featured"); v != "" {
		b, e := strconv.ParseBool(v)
		if e == nil {
			o.Featured = &b
		}
	}
	return o.Normalize()
}
func (s *Server) publicNews(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.PublicNews(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) publicFeaturedNews(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	b := true
	o.Featured = &b
	v, e := s.store.PublicNews(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) publicNewsOne(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.PublicNewsBySlug(r.Context(), chi.URLParam(r, "slug"))
	result(w, r, v, e)
}
func (s *Server) publicNewsCategories(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Categories(r.Context(), "news", true)
	result(w, r, v, e)
}
func (s *Server) publicTeam(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.PublicTeam(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) publicTeamOne(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.PublicTeamBySlug(r.Context(), chi.URLParam(r, "slug"))
	result(w, r, v, e)
}
func (s *Server) publicPracticeAreas(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Categories(r.Context(), "team", true)
	result(w, r, v, e)
}
func (s *Server) publicPracticeTeam(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	o.Category = chi.URLParam(r, "slug")
	v, e := s.store.PublicTeam(r.Context(), o)
	page(w, r, v, o, e)
}

type newsInput struct {
	domain.News
	TagIDs []uuid.UUID `json:"tagIds"`
}

func (s *Server) adminNews(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.AdminNews(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) adminNewsOne(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, e := s.store.NewsByID(r.Context(), id)
	result(w, r, v, e)
}
func (s *Server) createNews(w http.ResponseWriter, r *http.Request) {
	var in newsInput
	if !decode(w, r, &in) {
		return
	}
	in.CreatedBy = userFrom(r.Context()).ID
	if (in.Status == domain.NewsPublished || in.Status == domain.NewsScheduled) && !application.HasPermission(userFrom(r.Context()).Permissions, "news.publish") {
		fail(w, r, domain.ErrForbidden)
		return
	}
	if in.Status == "" {
		in.Status = domain.NewsDraft
	}
	e := s.content.CreateNews(r.Context(), &in.News, in.TagIDs)
	if e == nil {
		s.audit(r, "news.create", "news", in.ID.String(), nil, in.News)
	}
	created(w, r, in.News, e)
}
func (s *Server) updateNews(w http.ResponseWriter, r *http.Request) {
	var in newsInput
	if !decode(w, r, &in) {
		return
	}
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	in.ID = id
	before, _ := s.store.NewsByID(r.Context(), id)
	if (before.Status == domain.NewsPublished || in.Status == domain.NewsPublished || in.Status == domain.NewsScheduled) && !application.HasPermission(userFrom(r.Context()).Permissions, "news.publish") {
		fail(w, r, domain.ErrForbidden)
		return
	}
	e = s.content.UpdateNews(r.Context(), &in.News, in.TagIDs)
	if e == nil {
		s.audit(r, "news.update", "news", id.String(), before, in.News)
	}
	result(w, r, in.News, e)
}
func (s *Server) deleteNews(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, _ := strconv.Atoi(r.URL.Query().Get("version"))
	e = s.store.SoftDeleteNews(r.Context(), id, v)
	if e == nil {
		s.audit(r, "news.delete", "news", id.String(), nil, nil)
	}
	result(w, r, map[string]bool{"deleted": e == nil}, e)
}
func (s *Server) restoreNews(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, _ := strconv.Atoi(r.URL.Query().Get("version"))
	e = s.store.RestoreNews(r.Context(), id, v)
	result(w, r, map[string]bool{"restored": e == nil}, e)
}

type teamInput struct {
	domain.TeamMember
	PracticeAreaIDs []uuid.UUID `json:"practiceAreaIds"`
}

func (s *Server) adminTeam(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.AdminTeam(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) adminTeamOne(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, e := s.store.TeamByID(r.Context(), id)
	result(w, r, v, e)
}
func (s *Server) createTeam(w http.ResponseWriter, r *http.Request) {
	var in teamInput
	if !decode(w, r, &in) {
		return
	}
	in.CreatedBy = userFrom(r.Context()).ID
	if in.Status == "" {
		in.Status = domain.TeamDraft
	}
	e := s.content.CreateTeam(r.Context(), &in.TeamMember, in.PracticeAreaIDs)
	if e == nil {
		s.audit(r, "team.create", "team_member", in.ID.String(), nil, in.TeamMember)
	}
	created(w, r, in.TeamMember, e)
}
func (s *Server) updateTeam(w http.ResponseWriter, r *http.Request) {
	var in teamInput
	if !decode(w, r, &in) {
		return
	}
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	in.ID = id
	before, _ := s.store.TeamByID(r.Context(), id)
	e = s.content.UpdateTeam(r.Context(), &in.TeamMember, in.PracticeAreaIDs)
	if e == nil {
		s.audit(r, "team.update", "team_member", id.String(), before, in.TeamMember)
	}
	result(w, r, in.TeamMember, e)
}
func (s *Server) deleteTeam(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, _ := strconv.Atoi(r.URL.Query().Get("version"))
	e = s.store.SoftDeleteTeam(r.Context(), id, v)
	result(w, r, map[string]bool{"deleted": e == nil}, e)
}
func (s *Server) restoreTeam(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v, _ := strconv.Atoi(r.URL.Query().Get("version"))
	e = s.store.RestoreTeam(r.Context(), id, v)
	result(w, r, map[string]bool{"restored": e == nil}, e)
}
func (s *Server) adminCategories(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Categories(r.Context(), r.URL.Query().Get("type"), false)
	result(w, r, v, e)
}
func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	var v domain.Category
	if !decode(w, r, &v) {
		return
	}
	if v.DisplayOrder < 1 {
		v.DisplayOrder = 1
	}
	e := s.store.CreateCategory(r.Context(), &v)
	created(w, r, v, e)
}
func (s *Server) adminTags(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Tags(r.Context())
	result(w, r, v, e)
}
func (s *Server) createTag(w http.ResponseWriter, r *http.Request) {
	var v domain.Tag
	if !decode(w, r, &v) {
		return
	}
	if v.Slug == "" {
		v.Slug = application.Slug(v.Name)
	}
	e := s.store.CreateTag(r.Context(), &v)
	created(w, r, v, e)
}
func (s *Server) uploadMedia(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes+1024*1024)
	if e := r.ParseMultipartForm(s.cfg.MaxUploadBytes); e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	f, h, e := r.FormFile("file")
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	defer f.Close()
	v, e := s.media.Upload(r.Context(), f, h.Size, h.Filename, r.FormValue("altText"), r.FormValue("caption"), userFrom(r.Context()).ID)
	created(w, r, v, e)
}
func (s *Server) adminUsers(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.Users(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) createUser(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string      `json:"email"`
		Name     string      `json:"name"`
		Password string      `json:"password"`
		RoleIDs  []uuid.UUID `json:"roleIds"`
	}
	if !decode(w, r, &in) {
		return
	}
	hash, e := application.HashPassword(in.Password)
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	v := domain.User{Email: in.Email, Name: in.Name, PasswordHash: hash, Active: true}
	e = s.store.CreateUser(r.Context(), &v)
	if e == nil && len(in.RoleIDs) > 0 {
		e = s.store.AssignUserRoles(r.Context(), v.ID, in.RoleIDs)
	}
	if e == nil {
		s.audit(r, "user.create", "user", v.ID.String(), nil, v)
	}
	created(w, r, v, e)
}
func (s *Server) setUserActive(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	var in struct {
		Active bool `json:"isActive"`
	}
	if !decode(w, r, &in) {
		return
	}
	e = s.store.SetUserActive(r.Context(), id, in.Active)
	if e == nil {
		s.audit(r, "user.active_changed", "user", id.String(), nil, in)
	}
	result(w, r, map[string]bool{"isActive": in.Active}, e)
}
func (s *Server) assignUserRoles(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	var in struct {
		RoleIDs []uuid.UUID `json:"roleIds"`
	}
	if !decode(w, r, &in) {
		return
	}
	e = s.store.AssignUserRoles(r.Context(), id, in.RoleIDs)
	if e == nil {
		s.audit(r, "user.roles_changed", "user", id.String(), nil, in)
	}
	result(w, r, map[string]bool{"updated": e == nil}, e)
}
func (s *Server) adminRoles(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Roles(r.Context())
	result(w, r, v, e)
}
func (s *Server) createRole(w http.ResponseWriter, r *http.Request) {
	var v domain.Role
	if !decode(w, r, &v) {
		return
	}
	e := s.store.CreateRole(r.Context(), &v)
	if e == nil {
		s.audit(r, "role.create", "role", v.ID.String(), nil, v)
	}
	created(w, r, v, e)
}
func (s *Server) adminAuditLogs(w http.ResponseWriter, r *http.Request) {
	o := options(r)
	v, e := s.store.AuditLogs(r.Context(), o)
	page(w, r, v, o, e)
}
func (s *Server) adminRedirects(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.Redirects(r.Context())
	result(w, r, v, e)
}
func validateRedirect(v *domain.Redirect) error {
	if !strings.HasPrefix(v.SourcePath, "/") || !strings.HasPrefix(v.DestinationPath, "/") {
		return domain.ErrInvalid
	}
	if v.StatusCode == 0 {
		v.StatusCode = 301
	}
	if v.StatusCode != 301 && v.StatusCode != 302 && v.StatusCode != 307 && v.StatusCode != 308 {
		return domain.ErrInvalid
	}
	return nil
}
func (s *Server) createRedirect(w http.ResponseWriter, r *http.Request) {
	var v domain.Redirect
	if !decode(w, r, &v) {
		return
	}
	v.CreatedBy = userFrom(r.Context()).ID
	if e := validateRedirect(&v); e != nil {
		fail(w, r, e)
		return
	}
	loop, e := s.store.RedirectWouldLoop(r.Context(), v.SourcePath, v.DestinationPath, nil)
	if e == nil && loop {
		e = fmt.Errorf("%w: redirect loop", domain.ErrInvalid)
	}
	if e == nil {
		e = s.store.CreateRedirect(r.Context(), &v)
	}
	if e == nil {
		s.audit(r, "redirect.create", "redirect", v.ID.String(), nil, v)
	}
	created(w, r, v, e)
}
func (s *Server) updateRedirect(w http.ResponseWriter, r *http.Request) {
	var v domain.Redirect
	if !decode(w, r, &v) {
		return
	}
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	v.ID = id
	if e == nil {
		e = validateRedirect(&v)
	}
	if e == nil {
		var loop bool
		loop, e = s.store.RedirectWouldLoop(r.Context(), v.SourcePath, v.DestinationPath, &id)
		if loop {
			e = fmt.Errorf("%w: redirect loop", domain.ErrInvalid)
		}
	}
	if e == nil {
		e = s.store.UpdateRedirect(r.Context(), &v)
	}
	if e == nil {
		s.audit(r, "redirect.update", "redirect", id.String(), nil, v)
	}
	result(w, r, v, e)
}
func (s *Server) adminAPIClients(w http.ResponseWriter, r *http.Request) {
	v, e := s.store.APIClients(r.Context())
	result(w, r, v, e)
}
func (s *Server) createAPIClient(w http.ResponseWriter, r *http.Request) {
	var v domain.APIClient
	if !decode(w, r, &v) {
		return
	}
	secret, e := application.RandomToken(32)
	if e == nil {
		e = s.store.CreateAPIClient(r.Context(), &v, secret)
	}
	if e == nil {
		s.audit(r, "api_client.create", "api_client", v.ID.String(), nil, v)
	}
	if e != nil {
		fail(w, r, e)
		return
	}
	created(w, r, map[string]any{"client": v, "secret": secret}, nil)
}
func (s *Server) revokeAPIClient(w http.ResponseWriter, r *http.Request) {
	id, e := uuid.Parse(chi.URLParam(r, "id"))
	if e != nil {
		fail(w, r, domain.ErrInvalid)
		return
	}
	e = s.store.RevokeAPIClient(r.Context(), id)
	if e == nil {
		s.audit(r, "api_client.revoke", "api_client", id.String(), nil, nil)
	}
	result(w, r, map[string]bool{"revoked": e == nil}, e)
}
func (s *Server) audit(r *http.Request, action, typ, id string, before, after any) {
	u := userFrom(r.Context())
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	_ = s.store.CreateAudit(r.Context(), domain.AuditEvent{ActorID: &u.ID, ActorType: "user", Action: action, EntityType: typ, EntityID: id, RequestID: middleware.GetReqID(r.Context()), IP: ip, UserAgent: r.UserAgent(), Before: before, After: after})
}
func clientIP(r *http.Request) string {
	ip, _, e := net.SplitHostPort(r.RemoteAddr)
	if e != nil {
		return r.RemoteAddr
	}
	return ip
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if e := d.Decode(v); e != nil {
		writeError(w, r, 400, "VALIDATION_ERROR", "The request body is invalid", map[string]string{"body": e.Error()})
		return false
	}
	return true
}
func ok(w http.ResponseWriter, r *http.Request, v any) {
	writeJSON(w, 200, map[string]any{"data": v, "meta": map[string]string{"requestId": middleware.GetReqID(r.Context())}})
}
func created(w http.ResponseWriter, r *http.Request, v any, e error) {
	if e != nil {
		fail(w, r, e)
		return
	}
	writeJSON(w, 201, map[string]any{"data": v, "meta": map[string]string{"requestId": middleware.GetReqID(r.Context())}})
}
func result(w http.ResponseWriter, r *http.Request, v any, e error) {
	if e != nil {
		fail(w, r, e)
		return
	}
	ok(w, r, v)
}
func page[T any](w http.ResponseWriter, r *http.Request, v domain.Page[T], o domain.ListOptions, e error) {
	if e != nil {
		fail(w, r, e)
		return
	}
	pages := int((v.Total + int64(o.Limit) - 1) / int64(o.Limit))
	writeJSON(w, 200, map[string]any{"data": v.Items, "pagination": map[string]any{"page": o.Page, "limit": o.Limit, "total": v.Total, "totalPages": pages}, "meta": map[string]string{"requestId": middleware.GetReqID(r.Context())}})
}
func fail(w http.ResponseWriter, r *http.Request, e error) {
	switch {
	case errors.Is(e, domain.ErrNotFound):
		writeError(w, r, 404, "NOT_FOUND", "The requested resource was not found", nil)
	case errors.Is(e, domain.ErrConflict):
		writeError(w, r, 409, "CONFLICT", "The resource conflicts with current data or version", nil)
	case errors.Is(e, domain.ErrUnauthorized):
		writeError(w, r, 401, "UNAUTHORIZED", "Authentication is required", nil)
	case errors.Is(e, domain.ErrForbidden):
		writeError(w, r, 403, "FORBIDDEN", "You do not have permission for this action", nil)
	case errors.Is(e, domain.ErrInvalid):
		writeError(w, r, 422, "VALIDATION_ERROR", "The submitted data is invalid", nil)
	default:
		writeError(w, r, 500, "INTERNAL_ERROR", "An internal error occurred", nil)
	}
}
func writeError(w http.ResponseWriter, r *http.Request, status int, code, msg string, fields any) {
	writeJSONStatus(w, status, map[string]any{"error": map[string]any{"code": code, "message": msg, "fields": fields}, "meta": map[string]string{"requestId": middleware.GetReqID(r.Context())}})
}
func writeJSON(w http.ResponseWriter, status int, v any) { writeJSONStatus(w, status, v) }
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

type ipLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	limit  int
	window time.Duration
}

func newIPLimiter(n int, d time.Duration) *ipLimiter {
	return &ipLimiter{hits: map[string][]time.Time{}, limit: n, window: d}
}
func (l *ipLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cut := now.Add(-l.window)
	a := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cut) {
			a = append(a, t)
		}
	}
	if len(a) >= l.limit {
		l.hits[ip] = a
		return false
	}
	l.hits[ip] = append(a, now)
	return true
}
func (s *Server) loginRateLimit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.limiter.allow(r.RemoteAddr) {
			writeError(w, r, 429, "RATE_LIMITED", "Too many requests", nil)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var _ = fmt.Sprintf
var _ = strings.TrimSpace
