package main

import (
	"testing"
	"time"

	"entgo.io/ent/schema/field"
	"github.com/stretchr/testify/require"
)

func TestConvertValueBool(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want any
	}{
		{name: "sqlite stores integers", in: int64(1), want: true},
		{name: "sqlite zero is false", in: int64(0), want: false},
		{name: "already a bool", in: true, want: true},
		{name: "text true", in: []byte("true"), want: true},
		{name: "text zero", in: "0", want: false},
		{name: "null passes through", in: nil, want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := convertValue(field.TypeBool, tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestConvertValueBoolRejectsGarbage(t *testing.T) {
	t.Parallel()

	_, err := convertValue(field.TypeBool, "maybe")
	require.Error(t, err)
}

func TestConvertValueTime(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want time.Time
	}{
		{
			// The format Storyden's SQLite databases actually contain.
			name: "space separated with offset",
			in:   []byte("2026-07-23 13:00:24.3359653+02:00"),
			want: time.Date(2026, 7, 23, 13, 0, 24, 335965300, time.FixedZone("", 2*60*60)),
		},
		{
			name: "nanosecond precision utc offset",
			in:   "2026-08-07 22:28:59.536144901+00:00",
			want: time.Date(2026, 8, 7, 22, 28, 59, 536144901, time.UTC),
		},
		{
			name: "rfc3339",
			in:   "2026-08-07T22:28:59.536144901Z",
			want: time.Date(2026, 8, 7, 22, 28, 59, 536144901, time.UTC),
		},
		{
			name: "no fractional seconds",
			in:   "2026-08-07 22:28:59",
			want: time.Date(2026, 8, 7, 22, 28, 59, 0, time.UTC),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := convertValue(field.TypeTime, tc.in)
			require.NoError(t, err)

			parsed, ok := got.(time.Time)
			require.True(t, ok, "expected a time.Time, got %T", got)
			require.True(t, tc.want.Equal(parsed), "want %s, got %s", tc.want, parsed)
		})
	}
}

func TestConvertValueTimePassesThroughTimeValues(t *testing.T) {
	t.Parallel()

	now := time.Now()

	got, err := convertValue(field.TypeTime, now)
	require.NoError(t, err)
	require.Equal(t, now, got)
}

func TestConvertValueTimeEmptyBecomesNull(t *testing.T) {
	t.Parallel()

	got, err := convertValue(field.TypeTime, "")
	require.NoError(t, err)
	require.Nil(t, got)
}

func TestConvertValueTimeRejectsUnknownLayout(t *testing.T) {
	t.Parallel()

	_, err := convertValue(field.TypeTime, "23/07/2026")
	require.Error(t, err)
}

func TestConvertValueJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   any
		want any
	}{
		{
			// roles.permissions — every permission check reads this back via
			// sqljson.ValueContains, so it has to land in jsonb intact.
			name: "permission array",
			in:   []byte(`["CREATE_POST","READ_PUBLISHED_THREADS"]`),
			want: `["CREATE_POST","READ_PUBLISHED_THREADS"]`,
		},
		{name: "empty array", in: []byte("[]"), want: "[]"},
		{name: "empty object", in: "{}", want: "{}"},
		{name: "nested object", in: `{"editor":{"mode":"richtext"}}`, want: `{"editor":{"mode":"richtext"}}`},
		{name: "null passes through", in: nil, want: nil},
		{name: "empty string becomes null", in: "", want: nil},
		{name: "whitespace becomes null", in: []byte("   "), want: nil},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := convertValue(field.TypeJSON, tc.in)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
		})
	}
}

func TestConvertValueJSONRejectsInvalidPayloads(t *testing.T) {
	t.Parallel()

	_, err := convertValue(field.TypeJSON, []byte(`{"broken":`))
	require.ErrorContains(t, err, "not valid JSON")
}

func TestConvertValueStringAndEnum(t *testing.T) {
	t.Parallel()

	for _, typ := range []field.Type{field.TypeString, field.TypeEnum} {
		got, err := convertValue(typ, []byte("human"))
		require.NoError(t, err)
		require.Equal(t, "human", got)
	}
}

func TestConvertValueNumericPassesThrough(t *testing.T) {
	t.Parallel()

	got, err := convertValue(field.TypeInt, int64(42))
	require.NoError(t, err)
	require.Equal(t, int64(42), got)

	// roles.sort_key holds math.MaxFloat64 for the admin role.
	got, err = convertValue(field.TypeFloat64, 1.7976931348623157e+308)
	require.NoError(t, err)
	require.Equal(t, 1.7976931348623157e+308, got)
}

func TestInsertStatement(t *testing.T) {
	t.Parallel()

	require.Equal(t,
		`INSERT INTO "accounts" ("id", "handle") VALUES ($1, $2), ($3, $4)`,
		insertStatement("accounts", []string{"id", "handle"}, 2),
	)

	require.Equal(t,
		`INSERT INTO "roles" ("permissions") VALUES ($1)`,
		insertStatement("roles", []string{"permissions"}, 1),
	)
}

func TestEffectiveBatchSizeStaysUnderParameterLimit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		cols      int
		requested int
		want      int
	}{
		{name: "narrow table keeps requested size", cols: 4, requested: 200, want: 200},
		{name: "wide table is capped", cols: 500, requested: 200, want: 120},
		{name: "invalid request is clamped", cols: 4, requested: 0, want: 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := effectiveBatchSize(tc.cols, tc.requested)
			require.Equal(t, tc.want, got)
			require.LessOrEqual(t, got*tc.cols, maxParams)
		})
	}
}

func TestSQLitePathAcceptsURLsAndPaths(t *testing.T) {
	t.Parallel()

	require.Equal(t, "data/data.db", sqlitePath("sqlite://data/data.db"))
	require.Equal(t, "data/data.db?_pragma=foreign_keys(1)", sqlitePath("sqlite3://data/data.db?_pragma=foreign_keys(1)"))
	require.Equal(t, "docker/compose/data/data.db", sqlitePath("docker/compose/data/data.db"))
}

func TestPostgresDSNForcesSimpleProtocol(t *testing.T) {
	t.Parallel()

	dsn, err := postgresDSN("postgresql://user:pass@localhost:5432/storyden?sslmode=disable")
	require.NoError(t, err)
	require.Contains(t, dsn, "default_query_exec_mode=simple_protocol")
	require.Contains(t, dsn, "sslmode=disable")

	_, err = postgresDSN("sqlite://data/data.db")
	require.ErrorContains(t, err, "postgresql://")
}
