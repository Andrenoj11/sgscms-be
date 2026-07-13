package migrations

import (
	"context"
	_ "embed"

	"github.com/jackc/pgx/v5"
)

//go:embed 000001_init.up.sql
var initial string

func Up(ctx context.Context, databaseURL string) error {
	conn, e := pgx.Connect(ctx, databaseURL)
	if e != nil {
		return e
	}
	defer conn.Close(ctx)
	_, e = conn.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations(version bigint PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`)
	if e != nil {
		return e
	}
	var exists bool
	if e = conn.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=1)`).Scan(&exists); e != nil || exists {
		return e
	}
	tx, e := conn.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, initial, pgx.QueryExecModeSimpleProtocol); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `INSERT INTO schema_migrations(version)VALUES(1)`); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
