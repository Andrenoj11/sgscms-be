package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sgs-law-firm/cms/internal/domain"
)

type Store struct {
	pool          *pgxpool.Pool
	encryptionKey string
}

func New(ctx context.Context, url, encryptionKey string) (*Store, error) {
	p, e := pgxpool.New(ctx, url)
	if e != nil {
		return nil, e
	}
	if e = p.Ping(ctx); e != nil {
		p.Close()
		return nil, e
	}
	return &Store{pool: p, encryptionKey: encryptionKey}, nil
}
func (s *Store) Close()                       { s.pool.Close() }
func (s *Store) Ping(c context.Context) error { return s.pool.Ping(c) }
func (s *Store) BootstrapAdmin(c context.Context, email, name, passwordHash string) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	perms := []string{"news.read", "news.create", "news.update", "news.review", "news.publish", "news.delete", "team.read", "team.create", "team.update", "team.delete", "media.upload", "media.delete", "users.manage", "roles.manage", "audit.read", "redirects.manage"}
	for _, p := range perms {
		if _, e = tx.Exec(c, `INSERT INTO permissions(name)VALUES($1)ON CONFLICT(name)DO NOTHING`, p); e != nil {
			return e
		}
	}
	var roleID uuid.UUID
	e = tx.QueryRow(c, `INSERT INTO roles(name,description)VALUES('administrator','Full CMS access')ON CONFLICT(name)DO UPDATE SET description=excluded.description RETURNING id`).Scan(&roleID)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(c, `INSERT INTO role_permissions(role_id,permission_id)SELECT $1,id FROM permissions ON CONFLICT DO NOTHING`, roleID); e != nil {
		return e
	}
	var userID uuid.UUID
	e = tx.QueryRow(c, `INSERT INTO users(email,password_hash,name)VALUES(lower($1),$2,$3)ON CONFLICT(email)DO UPDATE SET name=excluded.name RETURNING id`, email, passwordHash, name).Scan(&userID)
	if e != nil {
		return e
	}
	if _, e = tx.Exec(c, `INSERT INTO user_roles(user_id,role_id)VALUES($1,$2)ON CONFLICT DO NOTHING`, userID, roleID); e != nil {
		return e
	}
	return tx.Commit(c)
}
func mapErr(e error) error {
	if errors.Is(e, pgx.ErrNoRows) {
		return domain.ErrNotFound
	}
	var pe *pgconn.PgError
	if errors.As(e, &pe) && pe.Code == "23505" {
		return domain.ErrConflict
	}
	return e
}

func (s *Store) UserByEmail(c context.Context, email string) (domain.User, error) {
	return s.user(c, `SELECT u.id,u.email,u.name,u.password_hash,u.is_active,u.last_login_at,u.created_at,u.updated_at,COALESCE(array_agg(DISTINCT p.name) FILTER(WHERE p.name IS NOT NULL),'{}') FROM users u LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN role_permissions rp ON rp.role_id=ur.role_id LEFT JOIN permissions p ON p.id=rp.permission_id WHERE lower(u.email)=lower($1) GROUP BY u.id`, email)
}
func (s *Store) UserBySessionToken(c context.Context, h string) (domain.User, error) {
	return s.user(c, `SELECT u.id,u.email,u.name,u.password_hash,u.is_active,u.last_login_at,u.created_at,u.updated_at,COALESCE(array_agg(DISTINCT p.name) FILTER(WHERE p.name IS NOT NULL),'{}') FROM user_sessions ss JOIN users u ON u.id=ss.user_id LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN role_permissions rp ON rp.role_id=ur.role_id LEFT JOIN permissions p ON p.id=rp.permission_id WHERE ss.token_hash=$1 AND ss.revoked_at IS NULL AND ss.expires_at>now() GROUP BY u.id`, h)
}
func (s *Store) user(c context.Context, q string, a any) (u domain.User, e error) {
	e = s.pool.QueryRow(c, q, a).Scan(&u.ID, &u.Email, &u.Name, &u.PasswordHash, &u.Active, &u.LastLoginAt, &u.CreatedAt, &u.UpdatedAt, &u.Permissions)
	return u, mapErr(e)
}
func (s *Store) CreateSession(c context.Context, v domain.Session) error {
	_, e := s.pool.Exec(c, `INSERT INTO user_sessions(id,user_id,token_hash,csrf_hash,expires_at) VALUES($1,$2,$3,$4,$5)`, v.ID, v.UserID, v.TokenHash, v.CSRFHash, v.ExpiresAt)
	return mapErr(e)
}
func (s *Store) DeleteSession(c context.Context, h string) error {
	_, e := s.pool.Exec(c, `UPDATE user_sessions SET revoked_at=now() WHERE token_hash=$1`, h)
	return e
}
func (s *Store) TouchLogin(c context.Context, id uuid.UUID) error {
	_, e := s.pool.Exec(c, `UPDATE users SET last_login_at=now(),failed_login_count=0,locked_until=NULL WHERE id=$1`, id)
	return e
}

func (s *Store) Categories(c context.Context, t string, public bool) ([]domain.Category, error) {
	q := `SELECT id,name,slug,type,COALESCE(description,''),display_order,is_active FROM categories`
	args := []any{}
	if t != "" {
		q += ` WHERE type=$1`
		args = append(args, t)
	} else {
		q += ` WHERE TRUE`
	}
	if public {
		q += ` AND is_active`
	}
	q += ` ORDER BY display_order,name`
	r, e := s.pool.Query(c, q, args...)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []domain.Category{}
	for r.Next() {
		var v domain.Category
		if e = r.Scan(&v.ID, &v.Name, &v.Slug, &v.Type, &v.Description, &v.DisplayOrder, &v.Active); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, r.Err()
}
func (s *Store) CreateCategory(c context.Context, v *domain.Category) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	e := s.pool.QueryRow(c, `INSERT INTO categories(id,name,slug,type,description,display_order,is_active)VALUES($1,$2,$3,$4,$5,$6,$7)RETURNING id`, v.ID, v.Name, v.Slug, v.Type, v.Description, v.DisplayOrder, v.Active).Scan(&v.ID)
	return mapErr(e)
}
func (s *Store) Tags(c context.Context) ([]domain.Tag, error) {
	r, e := s.pool.Query(c, `SELECT id,name,slug FROM tags ORDER BY name`)
	if e != nil {
		return nil, e
	}
	defer r.Close()
	out := []domain.Tag{}
	for r.Next() {
		var v domain.Tag
		if e = r.Scan(&v.ID, &v.Name, &v.Slug); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, r.Err()
}
func (s *Store) CreateTag(c context.Context, v *domain.Tag) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	_, e := s.pool.Exec(c, `INSERT INTO tags(id,name,slug)VALUES($1,$2,$3)`, v.ID, v.Name, v.Slug)
	return mapErr(e)
}
func (s *Store) CreateMedia(c context.Context, v *domain.Media) error {
	_, e := s.pool.Exec(c, `INSERT INTO media_files(id,file_name,original_name,mime_type,file_size,url,alt_text,caption,width,height,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, v.ID, v.FileName, v.OriginalName, v.MIMEType, v.FileSize, v.URL, v.AltText, v.Caption, v.Width, v.Height, v.CreatedBy)
	return mapErr(e)
}
func (s *Store) MediaReferencedByPublished(c context.Context, id uuid.UUID) (bool, error) {
	var x bool
	e := s.pool.QueryRow(c, `SELECT EXISTS(SELECT 1 FROM news WHERE featured_image_id=$1 AND status='published' AND deleted_at IS NULL) OR EXISTS(SELECT 1 FROM team_members WHERE photo_id=$1 AND status='active' AND deleted_at IS NULL)`, id).Scan(&x)
	return x, e
}
func (s *Store) RegisterNonce(c context.Context, key, nonce string, expires time.Time) error {
	_, e := s.pool.Exec(c, `INSERT INTO request_nonces(key_id,nonce,expires_at)VALUES($1,$2,$3)`, key, nonce, expires)
	return mapErr(e)
}
func (s *Store) APIClientSecret(c context.Context, key string) (string, []byte, []string, error) {
	var id string
	var secret string
	var perms []string
	e := s.pool.QueryRow(c, `SELECT key_id,pgp_sym_decrypt(secret_ciphertext,$2),allowed_permissions FROM api_clients WHERE key_id=$1 AND status='active' AND (expires_at IS NULL OR expires_at>now())`, key, s.encryptionKey).Scan(&id, &secret, &perms)
	return id, []byte(secret), perms, mapErr(e)
}
func (s *Store) CreateAudit(c context.Context, v domain.AuditEvent) error {
	before, _ := json.Marshal(v.Before)
	after, _ := json.Marshal(v.After)
	_, e := s.pool.Exec(c, `INSERT INTO audit_logs(actor_id,actor_type,action,entity_type,entity_id,request_id,ip_address,user_agent,previous_values,new_values)VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::inet,$8,$9,$10)`, v.ActorID, v.ActorType, v.Action, v.EntityType, v.EntityID, v.RequestID, v.IP, v.UserAgent, before, after)
	return e
}

func addTags(c context.Context, tx pgx.Tx, newsID uuid.UUID, tags []uuid.UUID) error {
	if _, e := tx.Exec(c, `DELETE FROM news_tags WHERE news_id=$1`, newsID); e != nil {
		return e
	}
	for _, id := range tags {
		if _, e := tx.Exec(c, `INSERT INTO news_tags(news_id,tag_id)VALUES($1,$2)`, newsID, id); e != nil {
			return e
		}
	}
	return nil
}
func addAreas(c context.Context, tx pgx.Tx, teamID uuid.UUID, areas []uuid.UUID) error {
	if _, e := tx.Exec(c, `DELETE FROM team_member_categories WHERE team_member_id=$1`, teamID); e != nil {
		return e
	}
	for _, id := range areas {
		if _, e := tx.Exec(c, `INSERT INTO team_member_categories(team_member_id,category_id)VALUES($1,$2)`, teamID, id); e != nil {
			return e
		}
	}
	return nil
}
func countQuery(c context.Context, p *pgxpool.Pool, q string, args ...any) (int64, error) {
	var n int64
	e := p.QueryRow(c, q, args...).Scan(&n)
	return n, e
}
func placeholders(start, count int) string {
	out := ""
	for i := 0; i < count; i++ {
		if i > 0 {
			out += ","
		}
		out += fmt.Sprintf("$%d", start+i)
	}
	return out
}
