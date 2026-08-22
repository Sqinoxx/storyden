package db

import (
	"context"
	"database/sql"
	"os"
	"strings"
	"testing"

	"ariga.io/atlas/sql/migrate"
	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"entgo.io/ent/dialect/sql/schema"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestAddOCRTextSearchIndex(t *testing.T) {
	dsn := os.Getenv("DATABASE_URL")
	if !strings.HasPrefix(strings.ToLower(dsn), "postgres") {
		t.Skip("requires a postgres DATABASE_URL")
	}

	sqlDB, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })

	if _, err := sqlDB.Exec(`DROP INDEX IF EXISTS assets_ocr_text_tsv_idx`); err != nil {
		t.Fatalf("failed to reset index for test: %v", err)
	}
	if _, err := sqlDB.Exec(`ALTER TABLE assets DROP COLUMN IF EXISTS ocr_text_tsv`); err != nil {
		t.Fatalf("failed to reset column for test: %v", err)
	}

	drv := entsql.OpenDB(dialect.Postgres, sqlDB)

	noop := schema.ApplyFunc(func(ctx context.Context, conn dialect.ExecQuerier, plan *migrate.Plan) error {
		return nil
	})
	applier := addOCRTextSearchIndex()(noop)

	if err := applier.Apply(context.Background(), drv, &migrate.Plan{}); err != nil {
		t.Fatalf("apply hook failed: %v", err)
	}

	var indexName string
	err = sqlDB.QueryRow(`SELECT indexname FROM pg_indexes WHERE indexname = 'assets_ocr_text_tsv_idx'`).Scan(&indexName)
	if err != nil {
		t.Fatalf("expected index to exist after apply: %v", err)
	}

	var dataType string
	err = sqlDB.QueryRow(`
		SELECT data_type FROM information_schema.columns
		WHERE table_name = 'assets' AND column_name = 'ocr_text_tsv'
	`).Scan(&dataType)
	if err != nil {
		t.Fatalf("expected generated column to exist: %v", err)
	}
	if dataType != "tsvector" {
		t.Fatalf("expected ocr_text_tsv to be tsvector, got %q", dataType)
	}

	// Exercise the exact generation expression and prefix-matching query form
	// node_search.ocrTextMatches uses (not needing a full asset row, since
	// assets.account_id has a required FK), including the case that motivated
	// prefix matching over websearch_to_tsquery: German compounds "anatomie"
	// into one lexeme with the following word ("Anatomieprüfung" stems to
	// "anatomiepruef"), so a whole-word query would silently miss it.
	var matched bool
	err = sqlDB.QueryRow(`
		SELECT to_tsvector('german', coalesce('Zahnmedizin Anatomieprüfung Wintersemester', ''))
			@@ (
				SELECT to_tsquery('german', string_agg(lexeme || ':*', ' & '))
				FROM unnest(tsvector_to_array(to_tsvector('german', 'anatomie'))) AS lexeme
			)
	`).Scan(&matched)
	if err != nil {
		t.Fatalf("failed to evaluate tsvector match: %v", err)
	}
	if !matched {
		t.Fatalf("expected prefix matching to find \"anatomie\" inside the compound word \"Anatomieprüfung\"")
	}

	// The hook runs on every startup, so it must tolerate the column and
	// index already existing.
	if err := applier.Apply(context.Background(), drv, &migrate.Plan{}); err != nil {
		t.Fatalf("second apply should be idempotent, got error: %v", err)
	}
}
