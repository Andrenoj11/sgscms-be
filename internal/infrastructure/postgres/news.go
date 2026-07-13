package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/sgs-law-firm/cms/internal/domain"
	"strings"
	"time"
)

const newsCols = `n.id,n.title,n.slug,COALESCE(n.excerpt,''),COALESCE(n.content,''),n.featured_image_id,n.category_id,n.status,n.is_featured,n.published_at,n.scheduled_at,n.created_by,COALESCE(u.name,''),COALESCE(n.meta_title,''),COALESCE(n.meta_description,''),COALESCE(n.canonical_url,''),COALESCE(n.legacy_id,''),COALESCE(n.legacy_path,''),n.view_count,n.version,n.created_at,n.updated_at,n.deleted_at`

type scanner interface{ Scan(...any) error }

func scanNews(r scanner) (v domain.News, e error) {
	e = r.Scan(&v.ID, &v.Title, &v.Slug, &v.Excerpt, &v.Content, &v.FeaturedImageID, &v.CategoryID, &v.Status, &v.Featured, &v.PublishedAt, &v.ScheduledAt, &v.CreatedBy, &v.AuthorName, &v.MetaTitle, &v.MetaDescription, &v.CanonicalURL, &v.LegacyID, &v.LegacyPath, &v.ViewCount, &v.Version, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return
}
func newsFilter(o domain.ListOptions, public bool) (string, []any) {
	w := []string{"n.deleted_at IS NULL"}
	args := []any{}
	add := func(clause string, v any) { args = append(args, v); w = append(w, fmt.Sprintf(clause, len(args))) }
	if public {
		w = append(w, "n.status='published' AND n.published_at<=now()")
	}
	if o.Search != "" {
		args = append(args, o.Search)
		i := len(args)
		w = append(w, fmt.Sprintf("(n.title ILIKE '%%'||$%d||'%%' OR n.slug ILIKE '%%'||$%d||'%%')", i, i))
	}
	if o.Category != "" {
		add("EXISTS(SELECT 1 FROM categories c WHERE c.id=n.category_id AND c.slug=$%d)", o.Category)
	}
	if o.Tag != "" {
		add("EXISTS(SELECT 1 FROM news_tags nt JOIN tags t ON t.id=nt.tag_id WHERE nt.news_id=n.id AND t.slug=$%d)", o.Tag)
	}
	if o.Status != "" && !public {
		add("n.status=$%d", o.Status)
	}
	if o.Featured != nil {
		add("n.is_featured=$%d", *o.Featured)
	}
	return strings.Join(w, " AND "), args
}
func (s *Store) newsList(c context.Context, o domain.ListOptions, public bool) (domain.Page[domain.News], error) {
	o = o.Normalize()
	where, args := newsFilter(o, public)
	total, e := countQuery(c, s.pool, "SELECT count(*) FROM news n WHERE "+where, args...)
	if e != nil {
		return domain.Page[domain.News]{}, e
	}
	order := "n.published_at DESC NULLS LAST,n.created_at DESC"
	args = append(args, o.Limit, (o.Page-1)*o.Limit)
	q := `SELECT ` + newsCols + ` FROM news n JOIN users u ON u.id=n.created_by WHERE ` + where + fmt.Sprintf(" ORDER BY %s LIMIT $%d OFFSET $%d", order, len(args)-1, len(args))
	rows, e := s.pool.Query(c, q, args...)
	if e != nil {
		return domain.Page[domain.News]{}, e
	}
	defer rows.Close()
	items := []domain.News{}
	for rows.Next() {
		v, e := scanNews(rows)
		if e != nil {
			return domain.Page[domain.News]{}, e
		}
		items = append(items, v)
	}
	if e = rows.Err(); e == nil {
		e = s.hydrateNews(c, items)
	}
	return domain.Page[domain.News]{Items: items, Total: total}, e
}
func (s *Store) PublicNews(c context.Context, o domain.ListOptions) (domain.Page[domain.News], error) {
	return s.newsList(c, o, true)
}
func (s *Store) AdminNews(c context.Context, o domain.ListOptions) (domain.Page[domain.News], error) {
	return s.newsList(c, o, false)
}
func (s *Store) PublicNewsBySlug(c context.Context, slug string) (domain.News, error) {
	v, e := scanNews(s.pool.QueryRow(c, `SELECT `+newsCols+` FROM news n JOIN users u ON u.id=n.created_by WHERE n.slug=$1 AND n.status='published' AND n.published_at<=now() AND n.deleted_at IS NULL`, slug))
	if e == nil {
		items := []domain.News{v}
		e = s.hydrateNews(c, items)
		v = items[0]
	}
	return v, mapErr(e)
}
func (s *Store) NewsByID(c context.Context, id uuid.UUID) (domain.News, error) {
	v, e := scanNews(s.pool.QueryRow(c, `SELECT `+newsCols+` FROM news n JOIN users u ON u.id=n.created_by WHERE n.id=$1`, id))
	if e == nil {
		items := []domain.News{v}
		e = s.hydrateNews(c, items)
		v = items[0]
	}
	return v, mapErr(e)
}

func (s *Store) hydrateNews(c context.Context, items []domain.News) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	index := map[uuid.UUID]int{}
	for i := range items {
		ids[i] = items[i].ID
		index[items[i].ID] = i
	}
	rows, e := s.pool.Query(c, `SELECT n.id,c.id,c.name,c.slug,c.type,COALESCE(c.description,''),c.display_order,c.is_active FROM news n JOIN categories c ON c.id=n.category_id WHERE n.id=ANY($1)`, ids)
	if e != nil {
		return e
	}
	for rows.Next() {
		var nid uuid.UUID
		var v domain.Category
		if e = rows.Scan(&nid, &v.ID, &v.Name, &v.Slug, &v.Type, &v.Description, &v.DisplayOrder, &v.Active); e != nil {
			rows.Close()
			return e
		}
		i := index[nid]
		items[i].Category = &v
	}
	rows.Close()
	rows, e = s.pool.Query(c, `SELECT nt.news_id,t.id,t.name,t.slug FROM news_tags nt JOIN tags t ON t.id=nt.tag_id WHERE nt.news_id=ANY($1) ORDER BY t.name`, ids)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var nid uuid.UUID
		var v domain.Tag
		if e = rows.Scan(&nid, &v.ID, &v.Name, &v.Slug); e != nil {
			return e
		}
		i := index[nid]
		items[i].Tags = append(items[i].Tags, v)
	}
	return rows.Err()
}
func (s *Store) CreateNews(c context.Context, n *domain.News, tags []uuid.UUID) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	if n.ID == uuid.Nil {
		n.ID = uuid.New()
	}
	e = tx.QueryRow(c, `INSERT INTO news(id,title,slug,excerpt,content,featured_image_id,category_id,status,is_featured,published_at,scheduled_at,created_by,meta_title,meta_description,canonical_url,legacy_id,legacy_path)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)RETURNING version,created_at,updated_at`, n.ID, n.Title, n.Slug, n.Excerpt, n.Content, n.FeaturedImageID, n.CategoryID, n.Status, n.Featured, n.PublishedAt, n.ScheduledAt, n.CreatedBy, n.MetaTitle, n.MetaDescription, n.CanonicalURL, n.LegacyID, n.LegacyPath).Scan(&n.Version, &n.CreatedAt, &n.UpdatedAt)
	if e != nil {
		return mapErr(e)
	}
	if e = addTags(c, tx, n.ID, tags); e != nil {
		return e
	}
	return tx.Commit(c)
}
func (s *Store) UpdateNews(c context.Context, n *domain.News, tags []uuid.UUID) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	tag, e := tx.Exec(c, `UPDATE news SET title=$2,slug=$3,excerpt=$4,content=$5,featured_image_id=$6,category_id=$7,status=$8,is_featured=$9,published_at=$10,scheduled_at=$11,meta_title=$12,meta_description=$13,canonical_url=$14,legacy_id=$15,legacy_path=$16,version=version+1,updated_at=now() WHERE id=$1 AND version=$17 AND deleted_at IS NULL`, n.ID, n.Title, n.Slug, n.Excerpt, n.Content, n.FeaturedImageID, n.CategoryID, n.Status, n.Featured, n.PublishedAt, n.ScheduledAt, n.MetaTitle, n.MetaDescription, n.CanonicalURL, n.LegacyID, n.LegacyPath, n.Version)
	if e != nil {
		return mapErr(e)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	if e = addTags(c, tx, n.ID, tags); e != nil {
		return e
	}
	n.Version++
	return tx.Commit(c)
}
func (s *Store) SoftDeleteNews(c context.Context, id uuid.UUID, version int) error {
	return versioned(c, s.pool, `UPDATE news SET deleted_at=now(),version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NULL`, id, version)
}
func (s *Store) RestoreNews(c context.Context, id uuid.UUID, version int) error {
	return versioned(c, s.pool, `UPDATE news SET deleted_at=NULL,version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NOT NULL`, id, version)
}
func versioned(c context.Context, q interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}, sql string, id uuid.UUID, v int) error {
	tag, e := q.Exec(c, sql, id, v)
	if e != nil {
		return mapErr(e)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	return nil
}
func (s *Store) PublishDueNews(c context.Context, now time.Time) (int64, error) {
	tag, e := s.pool.Exec(c, `UPDATE news SET status='published',published_at=scheduled_at,updated_at=$1,version=version+1 WHERE status='scheduled' AND scheduled_at<=$1 AND deleted_at IS NULL`, now)
	return tag.RowsAffected(), e
}
