package postgres

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"github.com/sgs-law-firm/cms/internal/domain"
	"strings"
)

const teamCols = `t.id,t.name,t.slug,COALESCE(t.academic_titles,''),COALESCE(t.position,''),COALESCE(t.short_bio,''),COALESCE(t.bio,''),t.photo_id,COALESCE(t.email,''),COALESCE(t.linkedin_url,''),t.display_order,t.is_featured,t.status,COALESCE(t.meta_title,''),COALESCE(t.meta_description,''),COALESCE(t.legacy_id,''),COALESCE(t.legacy_path,''),t.version,t.created_by,t.created_at,t.updated_at,t.deleted_at`

func scanTeam(r scanner) (v domain.TeamMember, e error) {
	e = r.Scan(&v.ID, &v.Name, &v.Slug, &v.AcademicTitles, &v.Position, &v.ShortBio, &v.Bio, &v.PhotoID, &v.Email, &v.LinkedInURL, &v.DisplayOrder, &v.Featured, &v.Status, &v.MetaTitle, &v.MetaDescription, &v.LegacyID, &v.LegacyPath, &v.Version, &v.CreatedBy, &v.CreatedAt, &v.UpdatedAt, &v.DeletedAt)
	return
}
func teamFilter(o domain.ListOptions, pub bool) (string, []any) {
	w := []string{"t.deleted_at IS NULL"}
	a := []any{}
	add := func(cl string, v any) { a = append(a, v); w = append(w, fmt.Sprintf(cl, len(a))) }
	if pub {
		w = append(w, "t.status='active'")
	}
	if o.Search != "" {
		a = append(a, o.Search)
		i := len(a)
		w = append(w, fmt.Sprintf("(t.name ILIKE '%%'||$%d||'%%' OR t.slug ILIKE '%%'||$%d||'%%')", i, i))
	}
	if o.Category != "" {
		add("EXISTS(SELECT 1 FROM team_member_categories tc JOIN categories c ON c.id=tc.category_id WHERE tc.team_member_id=t.id AND c.slug=$%d)", o.Category)
	}
	if o.Status != "" && !pub {
		add("t.status=$%d", o.Status)
	}
	if o.Featured != nil {
		add("t.is_featured=$%d", *o.Featured)
	}
	return strings.Join(w, " AND "), a
}
func (s *Store) teamList(c context.Context, o domain.ListOptions, pub bool) (domain.Page[domain.TeamMember], error) {
	o = o.Normalize()
	w, a := teamFilter(o, pub)
	total, e := countQuery(c, s.pool, "SELECT count(*) FROM team_members t WHERE "+w, a...)
	if e != nil {
		return domain.Page[domain.TeamMember]{}, e
	}
	a = append(a, o.Limit, (o.Page-1)*o.Limit)
	r, e := s.pool.Query(c, `SELECT `+teamCols+` FROM team_members t WHERE `+w+fmt.Sprintf(" ORDER BY t.display_order,t.name LIMIT $%d OFFSET $%d", len(a)-1, len(a)), a...)
	if e != nil {
		return domain.Page[domain.TeamMember]{}, e
	}
	defer r.Close()
	out := []domain.TeamMember{}
	for r.Next() {
		v, e := scanTeam(r)
		if e != nil {
			return domain.Page[domain.TeamMember]{}, e
		}
		out = append(out, v)
	}
	if e = r.Err(); e == nil {
		e = s.hydrateTeam(c, out)
	}
	return domain.Page[domain.TeamMember]{Items: out, Total: total}, e
}
func (s *Store) PublicTeam(c context.Context, o domain.ListOptions) (domain.Page[domain.TeamMember], error) {
	return s.teamList(c, o, true)
}
func (s *Store) AdminTeam(c context.Context, o domain.ListOptions) (domain.Page[domain.TeamMember], error) {
	return s.teamList(c, o, false)
}
func (s *Store) PublicTeamBySlug(c context.Context, slug string) (domain.TeamMember, error) {
	v, e := scanTeam(s.pool.QueryRow(c, `SELECT `+teamCols+` FROM team_members t WHERE t.slug=$1 AND t.status='active' AND t.deleted_at IS NULL`, slug))
	if e == nil {
		items := []domain.TeamMember{v}
		e = s.hydrateTeam(c, items)
		v = items[0]
	}
	return v, mapErr(e)
}
func (s *Store) TeamByID(c context.Context, id uuid.UUID) (domain.TeamMember, error) {
	v, e := scanTeam(s.pool.QueryRow(c, `SELECT `+teamCols+` FROM team_members t WHERE t.id=$1`, id))
	if e == nil {
		items := []domain.TeamMember{v}
		e = s.hydrateTeam(c, items)
		v = items[0]
	}
	return v, mapErr(e)
}

func (s *Store) hydrateTeam(c context.Context, items []domain.TeamMember) error {
	if len(items) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(items))
	index := map[uuid.UUID]int{}
	for i := range items {
		ids[i] = items[i].ID
		index[items[i].ID] = i
	}
	rows, e := s.pool.Query(c, `SELECT tc.team_member_id,c.id,c.name,c.slug,c.type,COALESCE(c.description,''),c.display_order,c.is_active FROM team_member_categories tc JOIN categories c ON c.id=tc.category_id WHERE tc.team_member_id=ANY($1) ORDER BY c.display_order,c.name`, ids)
	if e != nil {
		return e
	}
	defer rows.Close()
	for rows.Next() {
		var tid uuid.UUID
		var v domain.Category
		if e = rows.Scan(&tid, &v.ID, &v.Name, &v.Slug, &v.Type, &v.Description, &v.DisplayOrder, &v.Active); e != nil {
			return e
		}
		i := index[tid]
		items[i].PracticeAreas = append(items[i].PracticeAreas, v)
	}
	return rows.Err()
}
func (s *Store) CreateTeam(c context.Context, t *domain.TeamMember, areas []uuid.UUID) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	e = tx.QueryRow(c, `INSERT INTO team_members(id,name,slug,academic_titles,position,short_bio,bio,photo_id,email,linkedin_url,display_order,is_featured,status,meta_title,meta_description,legacy_id,legacy_path,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18)RETURNING version,created_at,updated_at`, t.ID, t.Name, t.Slug, t.AcademicTitles, t.Position, t.ShortBio, t.Bio, t.PhotoID, t.Email, t.LinkedInURL, t.DisplayOrder, t.Featured, t.Status, t.MetaTitle, t.MetaDescription, t.LegacyID, t.LegacyPath, t.CreatedBy).Scan(&t.Version, &t.CreatedAt, &t.UpdatedAt)
	if e != nil {
		return mapErr(e)
	}
	if e = addAreas(c, tx, t.ID, areas); e != nil {
		return e
	}
	return tx.Commit(c)
}
func (s *Store) UpdateTeam(c context.Context, t *domain.TeamMember, areas []uuid.UUID) error {
	tx, e := s.pool.Begin(c)
	if e != nil {
		return e
	}
	defer tx.Rollback(c)
	tag, e := tx.Exec(c, `UPDATE team_members SET name=$2,slug=$3,academic_titles=$4,position=$5,short_bio=$6,bio=$7,photo_id=$8,email=$9,linkedin_url=$10,display_order=$11,is_featured=$12,status=$13,meta_title=$14,meta_description=$15,legacy_id=$16,legacy_path=$17,version=version+1,updated_at=now() WHERE id=$1 AND version=$18 AND deleted_at IS NULL`, t.ID, t.Name, t.Slug, t.AcademicTitles, t.Position, t.ShortBio, t.Bio, t.PhotoID, t.Email, t.LinkedInURL, t.DisplayOrder, t.Featured, t.Status, t.MetaTitle, t.MetaDescription, t.LegacyID, t.LegacyPath, t.Version)
	if e != nil {
		return mapErr(e)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrConflict
	}
	if e = addAreas(c, tx, t.ID, areas); e != nil {
		return e
	}
	t.Version++
	return tx.Commit(c)
}
func (s *Store) SoftDeleteTeam(c context.Context, id uuid.UUID, v int) error {
	return versioned(c, s.pool, `UPDATE team_members SET deleted_at=now(),version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NULL`, id, v)
}
func (s *Store) RestoreTeam(c context.Context, id uuid.UUID, v int) error {
	return versioned(c, s.pool, `UPDATE team_members SET deleted_at=NULL,version=version+1 WHERE id=$1 AND version=$2 AND deleted_at IS NOT NULL`, id, v)
}
