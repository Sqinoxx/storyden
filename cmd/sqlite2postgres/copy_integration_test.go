package main

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	entmigrate "entgo.io/ent/dialect/sql/schema"
	"entgo.io/ent/dialect/sql/sqljson"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	entpredicate "github.com/Southclaws/storyden/internal/ent/predicate"
	ent_role "github.com/Southclaws/storyden/internal/ent/role"
	sd_schema "github.com/Southclaws/storyden/internal/ent/schema"
)

// TestCopyRoundTrip drives the real migration entrypoint against a throwaway
// Postgres database. It only runs when DATABASE_URL points at Postgres, which
// is how the postgres CI job is configured.
func TestCopyRoundTrip(t *testing.T) {
	adminURL := os.Getenv("DATABASE_URL")
	if !isPostgres(adminURL) {
		t.Skip("DATABASE_URL does not point at PostgreSQL")
	}

	ctx := context.Background()
	r := require.New(t)

	sourcePath := seedSource(t, ctx)
	targetURL := scratchDatabase(t, ctx, adminURL)

	target := openEnt(t, "pgx", mustSimpleProtocol(t, targetURL), dialect.Postgres)
	r.NoError(target.Schema.Create(ctx, entmigrate.WithDropIndex(true), entmigrate.WithDropColumn(true)))

	r.NoError(run(ctx, sourcePath, targetURL, 50, false, false))

	acc, err := target.Account.Query().Where(ent_account.Handle("testuser")).Only(ctx)
	r.NoError(err)

	r.Equal("Test User", acc.Name)
	r.True(acc.Admin, "bool column must survive the int64 to bool conversion")
	r.Equal(ent_account.KindHuman, acc.Kind)
	r.Equal(ent_account.VerifiedStatusEmail, acc.VerifiedStatus)
	r.Equal("richtext", acc.Metadata["editor"])
	r.Len(acc.Links, 1)
	r.Equal("https://example.com", acc.Links[0].URL)
	r.NotNil(acc.DeletedAt)
	r.WithinDuration(seedTime, acc.CreatedAt, time.Millisecond)

	other, err := target.Account.Query().Where(ent_account.Handle("plainuser")).Only(ctx)
	r.NoError(err)
	r.False(other.Admin)
	r.Nil(other.DeletedAt, "null timestamps must stay null")
	r.Empty(other.Metadata)

	role, err := target.Role.Query().Where(ent_role.Name("Admin")).Only(ctx)
	r.NoError(err)
	r.Equal([]string{"ADMINISTRATOR", "CREATE_POST"}, role.Permissions)
	r.Equal(math.MaxFloat64, role.SortKey)

	// The permission lookup the whole authorisation layer depends on. It compiles
	// to a jsonb containment query on Postgres and only works if permissions was
	// written as real jsonb rather than a quoted string.
	holders, err := target.Account.Query().
		Where(ent_account.HasRolesWith(entpredicate.Role(func(s *entsql.Selector) {
			s.Where(sqljson.ValueContains(ent_role.FieldPermissions, "ADMINISTRATOR"))
		}))).
		All(ctx)
	r.NoError(err)
	r.Len(holders, 1)
	r.Equal("testuser", holders[0].Handle)
}

func TestCopyRefusesNonEmptyTarget(t *testing.T) {
	adminURL := os.Getenv("DATABASE_URL")
	if !isPostgres(adminURL) {
		t.Skip("DATABASE_URL does not point at PostgreSQL")
	}

	ctx := context.Background()
	r := require.New(t)

	sourcePath := seedSource(t, ctx)
	targetURL := scratchDatabase(t, ctx, adminURL)

	target := openEnt(t, "pgx", mustSimpleProtocol(t, targetURL), dialect.Postgres)
	r.NoError(target.Schema.Create(ctx, entmigrate.WithDropIndex(true), entmigrate.WithDropColumn(true)))

	r.NoError(run(ctx, sourcePath, targetURL, 50, false, false))

	err := run(ctx, sourcePath, targetURL, 50, false, false)
	r.ErrorContains(err, "--truncate")

	r.NoError(run(ctx, sourcePath, targetURL, 50, true, false))

	n, err := target.Account.Query().Count(ctx)
	r.NoError(err)
	r.Equal(2, n, "truncate must replace rather than duplicate")
}

func TestCopyDryRunWritesNothing(t *testing.T) {
	adminURL := os.Getenv("DATABASE_URL")
	if !isPostgres(adminURL) {
		t.Skip("DATABASE_URL does not point at PostgreSQL")
	}

	ctx := context.Background()
	r := require.New(t)

	sourcePath := seedSource(t, ctx)
	targetURL := scratchDatabase(t, ctx, adminURL)

	target := openEnt(t, "pgx", mustSimpleProtocol(t, targetURL), dialect.Postgres)
	r.NoError(target.Schema.Create(ctx, entmigrate.WithDropIndex(true), entmigrate.WithDropColumn(true)))

	r.NoError(run(ctx, sourcePath, targetURL, 50, false, true))

	n, err := target.Account.Query().Count(ctx)
	r.NoError(err)
	r.Zero(n)
}

var seedTime = time.Date(2026, 7, 23, 13, 0, 24, 335965300, time.UTC)

// seedSource builds a SQLite database holding one row per awkward column type:
// booleans, nullable timestamps, jsonb objects, jsonb arrays, native enums and a
// float64 at the very edge of its range.
func seedSource(t *testing.T, ctx context.Context) string {
	t.Helper()
	r := require.New(t)

	path := filepath.Join(t.TempDir(), "source.db")

	client := openEnt(t, "sqlite", path+"?_pragma=foreign_keys(1)", dialect.SQLite)
	r.NoError(client.Schema.Create(ctx))

	deleted := seedTime.Add(time.Hour)
	signature := "hello"

	acc := client.Account.Create().
		SetHandle("testuser").
		SetName("Test User").
		SetBio("bio").
		SetSignature(signature).
		SetKind(ent_account.KindHuman).
		SetVerifiedStatus(ent_account.VerifiedStatusEmail).
		SetAdmin(true).
		SetLinks([]sd_schema.ExternalLink{{Text: "site", URL: "https://example.com"}}).
		SetMetadata(map[string]any{"editor": "richtext"}).
		SetCreatedAt(seedTime).
		SetUpdatedAt(seedTime).
		SetDeletedAt(deleted).
		SaveX(ctx)

	client.Account.Create().
		SetHandle("plainuser").
		SetName("Plain User").
		SetCreatedAt(seedTime).
		SetUpdatedAt(seedTime).
		SaveX(ctx)

	role := client.Role.Create().
		SetName("Admin").
		SetPermissions([]string{"ADMINISTRATOR", "CREATE_POST"}).
		SetSortKey(math.MaxFloat64).
		SetCreatedAt(seedTime).
		SetUpdatedAt(seedTime).
		SaveX(ctx)

	client.AccountRoles.Create().
		SetAccountID(acc.ID).
		SetRoleID(role.ID).
		SetBadge(true).
		SetCreatedAt(seedTime).
		SaveX(ctx)

	r.NoError(client.Close())

	return path
}

func openEnt(t *testing.T, driver, dsn, d string) *ent.Client {
	t.Helper()

	db, err := sql.Open(driver, dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	client := ent.NewClient(ent.Driver(entsql.OpenDB(d, db)))
	t.Cleanup(func() { _ = client.Close() })

	return client
}

// scratchDatabase creates an isolated database so the copier never truncates the
// shared CI database out from under the rest of the suite.
func scratchDatabase(t *testing.T, ctx context.Context, adminURL string) string {
	t.Helper()
	r := require.New(t)

	name := "sqlite2pg_" + strings.ToLower(xid.New().String())

	admin, err := sql.Open("pgx", adminURL)
	r.NoError(err)
	defer admin.Close()

	_, err = admin.ExecContext(ctx, fmt.Sprintf(`CREATE DATABASE %s`, quote(name)))
	r.NoError(err)

	t.Cleanup(func() {
		cleanup, err := sql.Open("pgx", adminURL)
		if err != nil {
			return
		}
		defer cleanup.Close()
		_, _ = cleanup.ExecContext(context.Background(), fmt.Sprintf(`DROP DATABASE IF EXISTS %s WITH (FORCE)`, quote(name)))
	})

	u, err := url.Parse(adminURL)
	r.NoError(err)
	u.Path = "/" + name

	return u.String()
}

func mustSimpleProtocol(t *testing.T, target string) string {
	t.Helper()

	dsn, err := postgresDSN(target)
	require.NoError(t, err)

	return dsn
}

func isPostgres(u string) bool {
	l := strings.ToLower(u)
	return strings.HasPrefix(l, "postgres://") || strings.HasPrefix(l, "postgresql://")
}
