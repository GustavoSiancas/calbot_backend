package db

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

const initialSchemaMigration = "000001_initial_schema"

//go:embed migrations/000001_initial_schema.up.sql
var initialSchemaSQL string

// ApplyInitialSchema applies the project's sole schema migration once per database.
func ApplyInitialSchema(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin schema migration: %w", err)
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(862104527)`); err != nil {
		return fmt.Errorf("lock schema migration: %w", err)
	}
	if _, err := tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name TEXT PRIMARY KEY, applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW())`); err != nil {
		return fmt.Errorf("create schema migration ledger: %w", err)
	}

	var applied bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name = $1)`, initialSchemaMigration).Scan(&applied); err != nil {
		return fmt.Errorf("check schema migration: %w", err)
	}
	if !applied {
		if _, err := tx.Exec(ctx, initialSchemaSQL); err != nil {
			return fmt.Errorf("apply initial schema: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (name) VALUES ($1)`, initialSchemaMigration); err != nil {
			return fmt.Errorf("record schema migration: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit schema migration: %w", err)
	}
	return nil
}
