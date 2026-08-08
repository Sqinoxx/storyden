// Command sqlite2postgres copies a Storyden SQLite database into an already
// migrated PostgreSQL database.
//
// The target schema must exist first:
//
//	DATABASE_URL=postgresql://... go run ./cmd/migrate
//	go run ./cmd/sqlite2postgres --source data/data.db --target postgresql://...
package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/signal"
	"sort"
	"strings"
	"text/tabwriter"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
	_ "github.com/glebarez/go-sqlite"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/Southclaws/storyden/internal/ent/migrate"
)

func main() {
	var (
		source   = flag.String("source", "docker/compose/data/data.db", "path to the source SQLite database (a sqlite:// URL is also accepted)")
		target   = flag.String("target", os.Getenv("DATABASE_URL"), "PostgreSQL connection URL of the target database (defaults to $DATABASE_URL)")
		batch    = flag.Int("batch", 200, "number of rows per multi-row insert")
		truncate = flag.Bool("truncate", false, "overwrite a target that already holds real data")
		dryRun   = flag.Bool("dry-run", false, "read, convert and insert everything but roll back instead of committing")
	)
	flag.Parse()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	if err := run(ctx, *source, *target, *batch, *truncate, *dryRun); err != nil {
		fmt.Fprintf(os.Stderr, "\nmigration failed: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, source, target string, batch int, truncate, dryRun bool) error {
	if target == "" {
		return fault.New("no target database specified, pass --target or set DATABASE_URL")
	}

	targetDSN, err := postgresDSN(target)
	if err != nil {
		return fault.Wrap(err)
	}

	src, err := sql.Open("sqlite", sqlitePath(source))
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to open source database"))
	}
	defer src.Close()

	if err := src.PingContext(ctx); err != nil {
		return fault.Wrap(err, fmsg.Withf("failed to connect to source database %q", source))
	}

	dst, err := sql.Open("pgx", targetDSN)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to open target database"))
	}
	defer dst.Close()

	if err := dst.PingContext(ctx); err != nil {
		return fault.Wrap(err, fmsg.With("failed to connect to target database"))
	}

	tables := migrate.Tables

	occupied, err := preflight(ctx, dst, tables)
	if err != nil {
		return fault.Wrap(err)
	}

	if len(occupied) > 0 && !truncate {
		return fault.Newf(
			"target database already holds data (%s), pass --truncate to overwrite it",
			strings.Join(occupied, ", "),
		)
	}

	sourceCounts, err := countRows(ctx, src, tables, true)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to count source rows"))
	}

	tx, err := dst.BeginTx(ctx, nil)
	if err != nil {
		return fault.Wrap(err, fmsg.With("failed to begin target transaction"))
	}
	defer tx.Rollback()

	// Disables foreign key triggers for this transaction. migrate.Tables is not
	// topologically sorted and cannot be — accounts and invitations reference
	// each other — so constraints have to stand down until every row is in.
	if _, err := tx.ExecContext(ctx, `SET LOCAL session_replication_role = replica`); err != nil {
		return fault.Wrap(err, fmsg.With("failed to disable foreign key enforcement, the target user needs superuser or replication rights"))
	}

	// This is a whole-database replacement, so the target always starts empty.
	// Beyond wiping real data on an explicit --truncate, this also clears the
	// bootstrap rows cmd/migrate leaves behind, which would otherwise collide on
	// their primary keys.
	if err := truncateAll(ctx, tx, tables); err != nil {
		return fault.Wrap(err)
	}

	c := &copier{src: src, tx: tx, batch: batch, tables: tables}

	results, err := c.run(ctx)
	if err != nil {
		return fault.Wrap(err)
	}

	if err := verify(results, sourceCounts); err != nil {
		return fault.Wrap(err)
	}

	if dryRun {
		if err := tx.Rollback(); err != nil {
			return fault.Wrap(err, fmsg.With("failed to roll back dry run"))
		}
	} else if err := tx.Commit(); err != nil {
		return fault.Wrap(err, fmsg.With("failed to commit"))
	}

	report(os.Stdout, results, dryRun)

	return nil
}

// bootstrapTables are populated by cmd/migrate itself — it boots the full
// application, which seeds a default row. Their presence says nothing about
// whether the target holds real data, so the emptiness check ignores them.
var bootstrapTables = map[string]struct{}{
	"settings": {},
}

// preflight checks that every table the current schema expects exists in the
// target and reports which of them already hold rows.
func preflight(ctx context.Context, dst *sql.DB, tables []*schema.Table) ([]string, error) {
	counts, err := countRows(ctx, dst, tables, false)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("target schema looks incomplete, run `go run ./cmd/migrate` against it first"))
	}

	occupied := []string{}
	for name, n := range counts {
		if _, ok := bootstrapTables[name]; ok {
			continue
		}
		if n > 0 {
			occupied = append(occupied, fmt.Sprintf("%s: %d", name, n))
		}
	}
	sort.Strings(occupied)

	return occupied, nil
}

// countRows counts every table. When skipMissing is set, tables absent from the
// database are omitted rather than failing — the source may predate a table.
func countRows(ctx context.Context, db *sql.DB, tables []*schema.Table, skipMissing bool) (map[string]int, error) {
	out := make(map[string]int, len(tables))

	for _, t := range tables {
		var n int
		err := db.QueryRowContext(ctx, fmt.Sprintf(`SELECT count(*) FROM %s`, quote(t.Name))).Scan(&n)
		if err != nil {
			if skipMissing {
				continue
			}
			return nil, fault.Wrap(err, fmsg.Withf("failed to count rows in %q", t.Name))
		}
		out[t.Name] = n
	}

	return out, nil
}

func truncateAll(ctx context.Context, tx *sql.Tx, tables []*schema.Table) error {
	names := make([]string, len(tables))
	for i, t := range tables {
		names[i] = quote(t.Name)
	}

	stmt := fmt.Sprintf(`TRUNCATE TABLE %s CASCADE`, strings.Join(names, ", "))
	if _, err := tx.ExecContext(ctx, stmt); err != nil {
		return fault.Wrap(err, fmsg.With("failed to truncate target tables"))
	}

	return nil
}

func verify(results []tableResult, sourceCounts map[string]int) error {
	mismatches := []string{}

	for _, r := range results {
		want, ok := sourceCounts[r.Name]
		if !ok {
			continue
		}
		if r.Copied != want {
			mismatches = append(mismatches, fmt.Sprintf("%s: source has %d rows, copied %d", r.Name, want, r.Copied))
		}
	}

	if len(mismatches) > 0 {
		return fault.Newf("row counts do not match, nothing was committed:\n  %s", strings.Join(mismatches, "\n  "))
	}

	return nil
}

func report(w *os.File, results []tableResult, dryRun bool) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	fmt.Fprintln(tw, "TABLE\tROWS\tNOTE")

	total := 0
	for _, r := range results {
		note := ""
		switch {
		case r.Skipped:
			note = r.SkipReason
		case len(r.DroppedCols) > 0:
			note = "columns missing in source: " + strings.Join(r.DroppedCols, ", ")
		}

		total += r.Copied
		fmt.Fprintf(tw, "%s\t%d\t%s\n", r.Name, r.Copied, note)
	}

	fmt.Fprintf(tw, "\t\t\n")
	fmt.Fprintf(tw, "TOTAL\t%d\t\n", total)
	tw.Flush()

	if dryRun {
		fmt.Fprintln(w, "\ndry run: everything was rolled back, no data was written")
	} else {
		fmt.Fprintln(w, "\nmigration committed successfully")
	}
}

// sqlitePath accepts either a plain filesystem path or a sqlite:// URL, matching
// what DATABASE_URL takes.
func sqlitePath(source string) string {
	for _, scheme := range []string{"sqlite://", "sqlite3://"} {
		if after, ok := strings.CutPrefix(source, scheme); ok {
			return after
		}
	}
	return source
}

// postgresDSN forces pgx into the simple protocol. Parameters are then sent as
// text literals and Postgres casts them itself, which is what lets plain Go
// strings land in the generated enum and jsonb columns without registering
// every custom type OID with pgx up front.
func postgresDSN(target string) (string, error) {
	u, err := url.Parse(target)
	if err != nil {
		return "", fault.Wrap(err, fmsg.With("failed to parse target database URL"))
	}

	switch u.Scheme {
	case "postgres", "postgresql":
	default:
		return "", fault.Newf("target must be a postgres:// or postgresql:// URL, got %q", u.Scheme)
	}

	q := u.Query()
	q.Set("default_query_exec_mode", "simple_protocol")
	u.RawQuery = q.Encode()

	return u.String(), nil
}
