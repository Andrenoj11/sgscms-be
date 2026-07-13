package postgres

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/sgs-law-firm/cms/internal/domain"
)

func (s *Store) Users(c context.Context, o domain.ListOptions) (domain.Page[domain.User], error) {
	o = o.Normalize()
	where := "TRUE"
	args := []any{}
	if o.Search != "" {
		args = append(args, o.Search)
		where = `(u.name ILIKE '%'||$1||'%' OR u.email ILIKE '%'||$1||'%')`
	}
	total, e := countQuery(c, s.pool, `SELECT count(*) FROM users u WHERE `+where, args...)
	if e != nil {
		return domain.Page[domain.User]{}, e
	}
	args = append(args, o.Limit, (o.Page-1)*o.Limit)
	rows, e := s.pool.Query(c, `SELECT u.id,u.email,u.name,u.is_active,u.last_login_at,u.created_at,u.updated_at,COALESCE(array_agg(DISTINCT p.name) FILTER(WHERE p.name IS NOT NULL),'{}') FROM users u LEFT JOIN user_roles ur ON ur.user_id=u.id LEFT JOIN role_permissions rp ON rp.role_id=ur.role_id LEFT JOIN permissions p ON p.id=rp.permission_id WHERE `+where+fmt.Sprintf(` GROUP BY u.id ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if e != nil {
		return domain.Page[domain.User]{}, e
	}
	defer rows.Close()
	items := []domain.User{}
	for rows.Next() {
		var v domain.User
		if e = rows.Scan(&v.ID, &v.Email, &v.Name, &v.Active, &v.LastLoginAt, &v.CreatedAt, &v.UpdatedAt, &v.Permissions); e != nil {
			return domain.Page[domain.User]{}, e
		}
		items = append(items, v)
	}
	return domain.Page[domain.User]{Items: items, Total: total}, rows.Err()
}
func (s *Store) CreateUser(c context.Context, u *domain.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	e := s.pool.QueryRow(c, `INSERT INTO users(id,email,password_hash,name,is_active)VALUES($1,lower($2),$3,$4,$5)RETURNING created_at,updated_at`, u.ID, u.Email, u.PasswordHash, u.Name, u.Active).Scan(&u.CreatedAt, &u.UpdatedAt)
	return mapErr(e)
}
func (s *Store) SetUserActive(c context.Context, id uuid.UUID, active bool) error {
	tag, e := s.pool.Exec(c, `UPDATE users SET is_active=$2,updated_at=now() WHERE id=$1`, id, active)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	if !active {
		_, _ = s.pool.Exec(c, `UPDATE user_sessions SET revoked_at=now() WHERE user_id=$1 AND revoked_at IS NULL`, id)
	}
	return nil
}
func (s *Store) AssignUserRoles(c context.Context, userID uuid.UUID, roles []uuid.UUID) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	if _, e = tx.Exec(c, `DELETE FROM user_roles WHERE user_id=$1`, userID); e != nil {
		return e
	}
	for _, id := range roles {
		if _, e = tx.Exec(c, `INSERT INTO user_roles(user_id,role_id)VALUES($1,$2)`, userID, id); e != nil {
			return mapErr(e)
		}
	}
	return tx.Commit(c)
}

func (s *Store) Roles(c context.Context) ([]domain.Role, error) {
	rows, e := s.pool.Query(c, `SELECT r.id,r.name,COALESCE(r.description,''),r.created_at,r.updated_at,COALESCE(array_agg(p.name ORDER BY p.name) FILTER(WHERE p.name IS NOT NULL),'{}') FROM roles r LEFT JOIN role_permissions rp ON rp.role_id=r.id LEFT JOIN permissions p ON p.id=rp.permission_id GROUP BY r.id ORDER BY r.name`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Role{}
	for rows.Next() {
		var v domain.Role
		if e = rows.Scan(&v.ID, &v.Name, &v.Description, &v.CreatedAt, &v.UpdatedAt, &v.Permissions); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) CreateRole(c context.Context, v *domain.Role) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	e = tx.QueryRow(c, `INSERT INTO roles(id,name,description)VALUES($1,$2,$3)RETURNING created_at,updated_at`, v.ID, v.Name, v.Description).Scan(&v.CreatedAt, &v.UpdatedAt)
	if e != nil {
		return mapErr(e)
	}
	for _, p := range v.Permissions {
		tag, e := tx.Exec(c, `INSERT INTO role_permissions(role_id,permission_id)SELECT $1,id FROM permissions WHERE name=$2`, v.ID, p)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("%w: unknown permission %s", domain.ErrInvalid, p)
		}
	}
	return tx.Commit(c)
}

func (s *Store) AuditLogs(c context.Context, o domain.ListOptions) (domain.Page[domain.AuditLog], error) {
	o = o.Normalize()
	where := "TRUE"
	args := []any{}
	if o.Search != "" {
		args = append(args, o.Search)
		where = `(action ILIKE '%'||$1||'%' OR entity_type ILIKE '%'||$1||'%')`
	}
	total, e := countQuery(c, s.pool, `SELECT count(*) FROM audit_logs WHERE `+where, args...)
	if e != nil {
		return domain.Page[domain.AuditLog]{}, e
	}
	args = append(args, o.Limit, (o.Page-1)*o.Limit)
	rows, e := s.pool.Query(c, `SELECT id,actor_id,actor_type,action,COALESCE(entity_type,''),COALESCE(entity_id,''),COALESCE(request_id,''),COALESCE(host(ip_address),''),COALESCE(user_agent,''),previous_values,new_values,created_at FROM audit_logs WHERE `+where+fmt.Sprintf(` ORDER BY created_at DESC LIMIT $%d OFFSET $%d`, len(args)-1, len(args)), args...)
	if e != nil {
		return domain.Page[domain.AuditLog]{}, e
	}
	defer rows.Close()
	items := []domain.AuditLog{}
	for rows.Next() {
		var v domain.AuditLog
		var before, after []byte
		if e = rows.Scan(&v.ID, &v.ActorID, &v.ActorType, &v.Action, &v.EntityType, &v.EntityID, &v.RequestID, &v.IP, &v.UserAgent, &before, &after, &v.CreatedAt); e != nil {
			return domain.Page[domain.AuditLog]{}, e
		}
		_ = json.Unmarshal(before, &v.PreviousValues)
		_ = json.Unmarshal(after, &v.NewValues)
		items = append(items, v)
	}
	return domain.Page[domain.AuditLog]{Items: items, Total: total}, rows.Err()
}

func (s *Store) Redirects(c context.Context) ([]domain.Redirect, error) {
	rows, e := s.pool.Query(c, `SELECT id,source_path,destination_path,status_code,is_active,COALESCE(created_by,'00000000-0000-0000-0000-000000000000'),created_at,updated_at FROM redirects ORDER BY source_path`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.Redirect{}
	for rows.Next() {
		var v domain.Redirect
		if e = rows.Scan(&v.ID, &v.SourcePath, &v.DestinationPath, &v.StatusCode, &v.Active, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) CreateRedirect(c context.Context, v *domain.Redirect) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	e := s.pool.QueryRow(c, `INSERT INTO redirects(id,source_path,destination_path,status_code,is_active,created_by)VALUES($1,$2,$3,$4,$5,$6)RETURNING created_at,updated_at`, v.ID, v.SourcePath, v.DestinationPath, v.StatusCode, v.Active, v.CreatedBy).Scan(&v.CreatedAt, &v.UpdatedAt)
	return mapErr(e)
}
func (s *Store) UpdateRedirect(c context.Context, v *domain.Redirect) error {
	tag, e := s.pool.Exec(c, `UPDATE redirects SET source_path=$2,destination_path=$3,status_code=$4,is_active=$5,updated_at=now() WHERE id=$1`, v.ID, v.SourcePath, v.DestinationPath, v.StatusCode, v.Active)
	if e != nil {
		return mapErr(e)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
func (s *Store) RedirectWouldLoop(c context.Context, source, dest string, exclude *uuid.UUID) (bool, error) {
	if strings.TrimSpace(source) == strings.TrimSpace(dest) {
		return true, nil
	}
	var loop bool
	e := s.pool.QueryRow(c, `WITH RECURSIVE chain(path) AS (SELECT $2::varchar UNION SELECT r.destination_path FROM redirects r JOIN chain c ON r.source_path=c.path WHERE r.is_active AND ($3::uuid IS NULL OR r.id<>$3)) SELECT EXISTS(SELECT 1 FROM chain WHERE path=$1)`, source, dest, exclude).Scan(&loop)
	return loop, e
}
func (s *Store) APIClients(c context.Context) ([]domain.APIClient, error) {
	rows, e := s.pool.Query(c, `SELECT id,client_id,key_id,status,allowed_permissions,expires_at,last_used_at,created_at,updated_at FROM api_clients ORDER BY created_at DESC`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []domain.APIClient{}
	for rows.Next() {
		var v domain.APIClient
		if e = rows.Scan(&v.ID, &v.ClientID, &v.KeyID, &v.Status, &v.Permissions, &v.ExpiresAt, &v.LastUsedAt, &v.CreatedAt, &v.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *Store) CreateAPIClient(c context.Context, v *domain.APIClient, secret string) error {
	if v.ID == uuid.Nil {
		v.ID = uuid.New()
	}
	if v.KeyID == "" {
		v.KeyID = "key_" + uuid.NewString()
	}
	if v.Status == "" {
		v.Status = "active"
	}
	e := s.pool.QueryRow(c, `INSERT INTO api_clients(id,client_id,key_id,secret_ciphertext,status,allowed_permissions,expires_at)VALUES($1,$2,$3,pgp_sym_encrypt($4,$5),$6,$7,$8)RETURNING created_at,updated_at`, v.ID, v.ClientID, v.KeyID, secret, s.encryptionKey, v.Status, v.Permissions, v.ExpiresAt).Scan(&v.CreatedAt, &v.UpdatedAt)
	return mapErr(e)
}
func (s *Store) RevokeAPIClient(c context.Context, id uuid.UUID) error {
	tag, e := s.pool.Exec(c, `UPDATE api_clients SET status='revoked',updated_at=now() WHERE id=$1`, id)
	if e != nil {
		return e
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
