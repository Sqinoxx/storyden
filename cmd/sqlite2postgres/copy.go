package main

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"entgo.io/ent/dialect/sql/schema"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fmsg"
)

// maxParams keeps multi-row inserts under the Postgres wire protocol limit of
// 65535 bound parameters per statement.
const maxParams = 60000

type tableResult struct {
	Name        string
	Copied      int
	Skipped     bool
	SkipReason  string
	DroppedCols []string
}

type copier struct {
	src    *sql.DB
	tx     *sql.Tx
	batch  int
	tables []*schema.Table
}

func (c *copier) run(ctx context.Context) ([]tableResult, error) {
	present, err := sourceTables(ctx, c.src)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	results := make([]tableResult, 0, len(c.tables))

	for _, t := range c.tables {
		if _, ok := present[t.Name]; !ok {
			results = append(results, tableResult{
				Name:       t.Name,
				Skipped:    true,
				SkipReason: "table not present in source database",
			})
			continue
		}

		res, err := c.copyTable(ctx, t)
		if err != nil {
			return nil, fault.Wrap(err, fmsg.Withf("failed to copy table %q", t.Name))
		}

		results = append(results, res)
	}

	return results, nil
}

func (c *copier) copyTable(ctx context.Context, t *schema.Table) (tableResult, error) {
	res := tableResult{Name: t.Name}

	sourceCols, err := sourceColumns(ctx, c.src, t.Name)
	if err != nil {
		return res, fault.Wrap(err)
	}

	cols := make([]*schema.Column, 0, len(t.Columns))
	for _, col := range t.Columns {
		if _, ok := sourceCols[col.Name]; !ok {
			res.DroppedCols = append(res.DroppedCols, col.Name)
			continue
		}
		cols = append(cols, col)
	}

	if len(cols) == 0 {
		res.Skipped = true
		res.SkipReason = "no columns in common with source"
		return res, nil
	}

	names := make([]string, len(cols))
	for i, col := range cols {
		names[i] = col.Name
	}

	rows, err := c.src.QueryContext(ctx, fmt.Sprintf(
		`SELECT %s FROM %s`,
		strings.Join(quoteAll(names), ", "),
		quote(t.Name),
	))
	if err != nil {
		return res, fault.Wrap(err, fmsg.With("failed to read source rows"))
	}
	defer rows.Close()

	batchRows := effectiveBatchSize(len(cols), c.batch)
	values := make([]any, 0, batchRows*len(cols))
	pending := 0

	flush := func() error {
		if pending == 0 {
			return nil
		}

		stmt := insertStatement(t.Name, names, pending)
		if _, err := c.tx.ExecContext(ctx, stmt, values...); err != nil {
			return fault.Wrap(err, fmsg.With("failed to insert rows into target"))
		}

		res.Copied += pending
		values = values[:0]
		pending = 0

		return nil
	}

	raw := make([]any, len(cols))
	ptrs := make([]any, len(cols))
	for i := range raw {
		ptrs[i] = &raw[i]
	}

	for rows.Next() {
		if err := rows.Scan(ptrs...); err != nil {
			return res, fault.Wrap(err, fmsg.With("failed to scan source row"))
		}

		for i, col := range cols {
			v, err := convertValue(col.Type, raw[i])
			if err != nil {
				return res, fault.Wrap(err, fmsg.Withf("failed to convert column %q", col.Name))
			}
			values = append(values, v)
		}

		pending++

		if pending >= batchRows {
			if err := flush(); err != nil {
				return res, fault.Wrap(err)
			}
		}
	}

	if err := rows.Err(); err != nil {
		return res, fault.Wrap(err, fmsg.With("failed while iterating source rows"))
	}

	if err := flush(); err != nil {
		return res, fault.Wrap(err)
	}

	return res, nil
}

func sourceTables(ctx context.Context, src *sql.DB) (map[string]struct{}, error) {
	rows, err := src.QueryContext(ctx, `SELECT name FROM sqlite_master WHERE type = 'table'`)
	if err != nil {
		return nil, fault.Wrap(err, fmsg.With("failed to list source tables"))
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fault.Wrap(err)
		}
		out[name] = struct{}{}
	}

	return out, fault.Wrap(rows.Err())
}

func sourceColumns(ctx context.Context, src *sql.DB, table string) (map[string]struct{}, error) {
	rows, err := src.QueryContext(ctx, fmt.Sprintf(`PRAGMA table_info(%s)`, quote(table)))
	if err != nil {
		return nil, fault.Wrap(err, fmsg.Withf("failed to inspect source table %q", table))
	}
	defer rows.Close()

	out := map[string]struct{}{}
	for rows.Next() {
		var (
			cid        int
			name       string
			typ        string
			notNull    int
			defaultVal any
			pk         int
		)
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultVal, &pk); err != nil {
			return nil, fault.Wrap(err)
		}
		out[name] = struct{}{}
	}

	return out, fault.Wrap(rows.Err())
}

func insertStatement(table string, cols []string, rows int) string {
	var b strings.Builder

	b.WriteString("INSERT INTO ")
	b.WriteString(quote(table))
	b.WriteString(" (")
	b.WriteString(strings.Join(quoteAll(cols), ", "))
	b.WriteString(") VALUES ")

	n := 1
	for r := range rows {
		if r > 0 {
			b.WriteString(", ")
		}
		b.WriteString("(")
		for c := range cols {
			if c > 0 {
				b.WriteString(", ")
			}
			b.WriteString("$")
			b.WriteString(strconv.Itoa(n))
			n++
		}
		b.WriteString(")")
	}

	return b.String()
}

func effectiveBatchSize(cols, requested int) int {
	if requested < 1 {
		requested = 1
	}
	if cols < 1 {
		return requested
	}
	if limit := maxParams / cols; requested > limit {
		if limit < 1 {
			return 1
		}
		return limit
	}
	return requested
}

func quote(ident string) string {
	return `"` + strings.ReplaceAll(ident, `"`, `""`) + `"`
}

func quoteAll(idents []string) []string {
	out := make([]string, len(idents))
	for i, s := range idents {
		out[i] = quote(s)
	}
	return out
}
