package node_search

import (
	"context"
	"strings"

	"entgo.io/ent/dialect/sql"
	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/jmoiron/sqlx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/role/role_hydrate"
	"github.com/Southclaws/storyden/app/resources/library"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/app/resources/visibility"
	"github.com/Southclaws/storyden/internal/ent"
	ent_account "github.com/Southclaws/storyden/internal/ent/account"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/ent/node"
	"github.com/Southclaws/storyden/internal/ent/predicate"
	ent_tag "github.com/Southclaws/storyden/internal/ent/tag"
)

// ocrTextSearchColumn is a generated tsvector column (see
// internal/infrastructure/db.addOCRTextSearchIndex) derived from
// assets.ocr_text. Matching against it instead of running ILIKE directly on
// ocr_text lets Postgres use a GIN index without ever reading the — large,
// TOASTed — raw text at query time.
const ocrTextSearchColumn = "ocr_text_tsv"

// ocrTextMatches matches search results against a document's extracted text.
// On Postgres this queries ocrTextSearchColumn so the GIN index can serve it;
// on SQLite/libSQL, which have no tsvector/GIN equivalent and no generated
// column to query, it falls back to a case-insensitive substring scan.
//
// The query terms are matched as lexeme prefixes rather than with
// websearch_to_tsquery's whole-word matching: German freely compounds words
// ("Anatomieprüfung" stems to one lexeme, "anatomiepruef"), so a search for
// "anatomie" would otherwise miss it entirely. Prefix matching restores
// close-to-substring recall while still hitting the GIN index.
func ocrTextMatches(driverName, searchTerm string) predicate.Asset {
	if driverName != "pgx" {
		return ent_asset.OcrTextContainsFold(searchTerm)
	}

	return predicate.Asset(func(s *sql.Selector) {
		// sql.ExprP's "?" placeholders are not rewritten for Postgres (which
		// needs "$1"), so the argument is bound through sql.P/Arg instead,
		// which is dialect-aware.
		s.Where(sql.P(func(b *sql.Builder) {
			b.WriteString(ocrTextSearchColumn + ` @@ (
				SELECT to_tsquery('german', string_agg(lexeme || ':*', ' & '))
				FROM unnest(tsvector_to_array(to_tsvector('german', `)
			b.Arg(searchTerm)
			b.WriteString(`))) AS lexeme
			)`)
		}))
	})
}

type Search interface {
	Search(ctx context.Context, params pagination.Parameters, opts ...Option) (*pagination.Result[*library.Node], error)
}

type query struct {
	nameContains    string
	contentContains string
	visibility      []visibility.Visibility
	authors         []account.AccountID
	tags            []tag_ref.Name
}

type Option func(*query)

func WithNameContains(s string) Option {
	return func(q *query) {
		q.nameContains = s
	}
}

func WithContentContains(s string) Option {
	return func(q *query) {
		q.contentContains = s
	}
}

func WithVisibility(v []visibility.Visibility) Option {
	return func(q *query) {
		q.visibility = v
	}
}

func WithAuthors(ids ...account.AccountID) Option {
	return func(q *query) {
		q.authors = ids
	}
}

func WithTags(names ...tag_ref.Name) Option {
	return func(q *query) {
		q.tags = names
	}
}

type service struct {
	db          *ent.Client
	raw         *sqlx.DB
	roleQuerier *role_hydrate.Hydrator
}

func New(db *ent.Client, raw *sqlx.DB, roleQuerier *role_hydrate.Hydrator) Search {
	return &service{
		db:          db,
		raw:         raw,
		roleQuerier: roleQuerier,
	}
}

func (s *service) Search(ctx context.Context, params pagination.Parameters, opts ...Option) (*pagination.Result[*library.Node], error) {
	q := &query{}

	for _, fn := range opts {
		fn(q)
	}

	baseQuery := s.db.Node.Query().Where(
		node.VisibilityEQ(node.VisibilityPublished),
		node.DeletedAtIsNil(),
	)

	nameContains := strings.TrimSpace(q.nameContains)
	contentContains := strings.TrimSpace(q.contentContains)

	if nameContains != "" || contentContains != "" {
		searchTerm := contentContains
		if searchTerm == "" {
			searchTerm = nameContains
		}

		// Only OR in predicates for terms that were actually supplied — an
		// empty ContainsFold("") degenerates to LIKE '%%', which matches
		// every row, so it must never be included unconditionally.
		predicates := []predicate.Node{
			node.HasAssetsWith(ocrTextMatches(s.raw.DriverName(), searchTerm)),
		}
		if nameContains != "" {
			predicates = append(predicates, node.NameContainsFold(nameContains))
		}
		if contentContains != "" {
			predicates = append(predicates, node.ContentContainsFold(contentContains))
		}

		baseQuery = baseQuery.Where(node.Or(predicates...))
	}

	if len(q.authors) > 0 {
		authorIDs := dt.Map(q.authors, func(id account.AccountID) xid.ID {
			return xid.ID(id)
		})
		baseQuery = baseQuery.Where(node.HasOwnerWith(ent_account.IDIn(authorIDs...)))
	}

	if len(q.tags) > 0 {
		for _, tag := range q.tags {
			baseQuery = baseQuery.Where(node.HasTagsWith(ent_tag.NameEQ(tag.String())))
		}
	}

	total, err := baseQuery.Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	query := baseQuery.
		WithOwner().
		WithNodes(func(cq *ent.NodeQuery) {
			cq.WithOwner()
		}).
		WithPrimaryImage().
		Order(node.ByUpdatedAt(sql.OrderDesc()), node.ByCreatedAt(sql.OrderDesc())).
		Limit(params.Limit()).
		Offset(params.Offset())

	r, err := query.All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	roleTargets := make([]*ent.Account, 0, len(r)*2)
	for _, n := range r {
		roleTargets = append(roleTargets, library.RoleHydrationTargetsFromNode(n)...)
	}
	if err := s.roleQuerier.HydrateRoleEdges(ctx, roleTargets...); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	nodes, err := dt.MapErr(r, library.MapNode(true, nil))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	result := pagination.NewPageResult(params, total, nodes)

	return &result, nil
}
