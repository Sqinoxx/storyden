package listing_repo

import (
	"context"
	stdsql "database/sql"
	"errors"
	"fmt"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/role/role_hydrate"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/listing"
	"github.com/Southclaws/storyden/app/resources/mark"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/internal/ent"
	ent_listing "github.com/Southclaws/storyden/internal/ent/listing"
	"github.com/Southclaws/storyden/internal/ent/listingfavourite"
	"github.com/Southclaws/storyden/internal/ent/listingoffer"
)

type Repository struct {
	db    *ent.Client
	roles *role_hydrate.Hydrator
}

func New(db *ent.Client, roles *role_hydrate.Hydrator) *Repository {
	return &Repository{db: db, roles: roles}
}

type Option func(*ent.ListingMutation)

func WithTitle(v string) Option {
	return func(m *ent.ListingMutation) { m.SetTitle(v) }
}

func WithDescription(v string) Option {
	return func(m *ent.ListingMutation) { m.SetDescription(v) }
}

func WithPrice(cents int) Option {
	return func(m *ent.ListingMutation) { m.SetPriceCents(cents) }
}

func WithPriceType(v listing.PriceType) Option {
	return func(m *ent.ListingMutation) { m.SetPriceType(ent_listing.PriceType(v.String())) }
}

func WithCategory(v string) Option {
	return func(m *ent.ListingMutation) { m.SetCategory(v) }
}

func WithCondition(v listing.Condition) Option {
	return func(m *ent.ListingMutation) { m.SetCondition(ent_listing.Condition(v.String())) }
}

func WithStatus(v listing.Status) Option {
	return func(m *ent.ListingMutation) { m.SetStatus(ent_listing.Status(v.String())) }
}

func WithLocation(v opt.Optional[string]) Option {
	return func(m *ent.ListingMutation) {
		if s, ok := v.Get(); ok && s != "" {
			m.SetLocation(s)
		} else {
			m.ClearLocation()
		}
	}
}

func WithExpiresAt(t time.Time) Option {
	return func(m *ent.ListingMutation) {
		m.SetExpiresAt(t)
		m.ClearExpiryNotifiedAt()
	}
}

func WithAssets(ids []asset.AssetID) Option {
	return func(m *ent.ListingMutation) {
		m.ClearAssets()
		m.AddAssetIDs(ids...)
		m.SetAssetOrder(dt.Map(ids, func(id asset.AssetID) string { return id.String() }))
	}
}

func (r *Repository) Create(ctx context.Context, owner account.AccountID, title string, expiresAt time.Time, opts ...Option) (*listing.Listing, error) {
	id := xid.New()
	create := r.db.Listing.Create()
	mutation := create.Mutation()

	mutation.SetID(id)
	mutation.SetAccountID(xid.ID(owner))
	mutation.SetTitle(title)
	mutation.SetSlug(fmt.Sprintf("%s-%s", id, mark.Slugify(title)))
	mutation.SetDescription("")
	mutation.SetExpiresAt(expiresAt)

	for _, fn := range opts {
		fn(mutation)
	}

	err := create.Exec(ctx)
	if err != nil {
		if ent.IsConstraintError(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.AlreadyExists))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.Get(ctx, listing.NewID(listing.ID(id)))
}

func (r *Repository) Update(ctx context.Context, id listing.ID, opts ...Option) (*listing.Listing, error) {
	update := r.db.Listing.UpdateOneID(xid.ID(id))
	mutation := update.Mutation()

	for _, fn := range opts {
		fn(mutation)
	}

	if title, ok := mutation.Title(); ok {
		mutation.SetSlug(fmt.Sprintf("%s-%s", xid.ID(id), mark.Slugify(title)))
	}

	err := update.Exec(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.Get(ctx, listing.NewID(id))
}

func (r *Repository) Delete(ctx context.Context, id listing.ID) error {
	err := r.db.Listing.UpdateOneID(xid.ID(id)).SetDeletedAt(time.Now()).Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (r *Repository) query() *ent.ListingQuery {
	return r.db.Listing.Query().
		Where(ent_listing.DeletedAtIsNil()).
		WithOwner().
		WithAssets()
}

func (r *Repository) mapMany(ctx context.Context, rows []*ent.Listing) ([]*listing.Listing, error) {
	owners := make([]*ent.Account, 0, len(rows))
	for _, row := range rows {
		if row.Edges.Owner != nil {
			owners = append(owners, row.Edges.Owner)
		}
	}
	if err := r.roles.HydrateRoleEdges(ctx, owners...); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.MapErr(rows, listing.Map)
}

func (r *Repository) Get(ctx context.Context, key listing.QueryKey) (*listing.Listing, error) {
	row, err := r.query().Where(key.Predicate()).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	items, err := r.mapMany(ctx, []*ent.Listing{row})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return items[0], nil
}

type Filter func(*ent.ListingQuery)

func WithQuery(q string) Filter {
	return func(lq *ent.ListingQuery) {
		lq.Where(ent_listing.Or(
			ent_listing.TitleContainsFold(q),
			ent_listing.DescriptionContainsFold(q),
			ent_listing.LocationContainsFold(q),
		))
	}
}

func WithCategories(v ...string) Filter {
	return func(lq *ent.ListingQuery) { lq.Where(ent_listing.CategoryIn(v...)) }
}

func WithConditions(v ...listing.Condition) Filter {
	return func(lq *ent.ListingQuery) {
		lq.Where(ent_listing.ConditionIn(dt.Map(v, func(c listing.Condition) ent_listing.Condition { return ent_listing.Condition(c.String()) })...))
	}
}

func WithPriceTypes(v ...listing.PriceType) Filter {
	return func(lq *ent.ListingQuery) {
		lq.Where(ent_listing.PriceTypeIn(dt.Map(v, func(c listing.PriceType) ent_listing.PriceType { return ent_listing.PriceType(c.String()) })...))
	}
}

func WithStatuses(v ...listing.Status) Filter {
	return func(lq *ent.ListingQuery) {
		lq.Where(ent_listing.StatusIn(dt.Map(v, func(c listing.Status) ent_listing.Status { return ent_listing.Status(c.String()) })...))
	}
}

func WithPriceMin(cents int) Filter {
	return func(lq *ent.ListingQuery) { lq.Where(ent_listing.PriceCentsGTE(cents)) }
}

func WithPriceMax(cents int) Filter {
	return func(lq *ent.ListingQuery) { lq.Where(ent_listing.PriceCentsLTE(cents)) }
}

func WithOwner(id account.AccountID) Filter {
	return func(lq *ent.ListingQuery) { lq.Where(ent_listing.AccountID(xid.ID(id))) }
}

func WithFavouritedBy(id account.AccountID) Filter {
	return func(lq *ent.ListingQuery) {
		lq.Where(ent_listing.HasFavouritesWith(listingfavourite.AccountID(xid.ID(id))))
	}
}

func WithActiveAt(now time.Time) Filter {
	return func(lq *ent.ListingQuery) { lq.Where(ent_listing.ExpiresAtGT(now)) }
}

type Sort string

const (
	SortNewest    Sort = "newest"
	SortPriceAsc  Sort = "price_asc"
	SortPriceDesc Sort = "price_desc"
)

func NewSort(s string) Sort {
	switch Sort(s) {
	case SortPriceAsc, SortPriceDesc:
		return Sort(s)
	default:
		return SortNewest
	}
}

func (r *Repository) List(ctx context.Context, params pagination.Parameters, sort Sort, filters ...Filter) (*pagination.Result[*listing.Listing], error) {
	q := r.query()
	for _, fn := range filters {
		fn(q)
	}

	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	switch sort {
	case SortPriceAsc:
		q.Order(ent_listing.ByPriceCents(), ent_listing.ByCreatedAt(sql.OrderDesc()))
	case SortPriceDesc:
		q.Order(ent_listing.ByPriceCents(sql.OrderDesc()), ent_listing.ByCreatedAt(sql.OrderDesc()))
	default:
		q.Order(ent_listing.ByCreatedAt(sql.OrderDesc()))
	}

	rows, err := q.Limit(params.Limit()).Offset(params.Offset()).All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	items, err := r.mapMany(ctx, rows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	result := pagination.NewPageResult(params, total, items)

	return &result, nil
}

func (r *Repository) PriceBounds(ctx context.Context, filters ...Filter) (int, int, error) {
	q := r.query()
	for _, fn := range filters {
		fn(q)
	}

	var rows []struct {
		Min *int `json:"min"`
		Max *int `json:"max"`
	}
	err := q.Aggregate(ent.Min(ent_listing.FieldPriceCents), ent.Max(ent_listing.FieldPriceCents)).Scan(ctx, &rows)
	if err != nil {
		return 0, 0, fault.Wrap(err, fctx.With(ctx))
	}
	if len(rows) == 0 || rows[0].Min == nil || rows[0].Max == nil {
		return 0, 0, nil
	}

	return *rows[0].Min, *rows[0].Max, nil
}

func (r *Repository) ListExpiringUnnotified(ctx context.Context, now time.Time, before time.Time) ([]*listing.Listing, error) {
	rows, err := r.query().
		Where(
			ent_listing.ExpiresAtGT(now),
			ent_listing.ExpiresAtLTE(before),
			ent_listing.ExpiryNotifiedAtIsNil(),
			ent_listing.StatusNEQ(ent_listing.StatusSold),
		).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.mapMany(ctx, rows)
}

func (r *Repository) MarkExpiryNotified(ctx context.Context, id listing.ID, at time.Time) error {
	err := r.db.Listing.UpdateOneID(xid.ID(id)).SetExpiryNotifiedAt(at).Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (r *Repository) UpsertOffer(ctx context.Context, id listing.ID, buyer account.AccountID, amountCents int, message string) (*listing.Offer, error) {
	err := r.db.ListingOffer.Create().
		SetListingID(xid.ID(id)).
		SetAccountID(xid.ID(buyer)).
		SetAmountCents(amountCents).
		SetMessage(message).
		OnConflictColumns(listingoffer.FieldListingID, listingoffer.FieldAccountID).
		Update(func(u *ent.ListingOfferUpsert) {
			u.UpdateAmountCents()
			u.UpdateMessage()
			u.UpdateUpdatedAt()
		}).
		Exec(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	offer, ok, err := r.GetOffer(ctx, id, buyer)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if !ok {
		return nil, fault.New("offer missing after upsert", fctx.With(ctx))
	}

	return offer, nil
}

func (r *Repository) offerQuery() *ent.ListingOfferQuery {
	return r.db.ListingOffer.Query().WithBuyer()
}

func (r *Repository) mapOffers(ctx context.Context, rows []*ent.ListingOffer) ([]*listing.Offer, error) {
	buyers := make([]*ent.Account, 0, len(rows))
	for _, row := range rows {
		if row.Edges.Buyer != nil {
			buyers = append(buyers, row.Edges.Buyer)
		}
	}
	if err := r.roles.HydrateRoleEdges(ctx, buyers...); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.MapErr(rows, listing.MapOffer)
}

func (r *Repository) GetOffer(ctx context.Context, id listing.ID, buyer account.AccountID) (*listing.Offer, bool, error) {
	row, err := r.offerQuery().
		Where(listingoffer.ListingID(xid.ID(id)), listingoffer.AccountID(xid.ID(buyer))).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, false, nil
		}
		return nil, false, fault.Wrap(err, fctx.With(ctx))
	}

	offers, err := r.mapOffers(ctx, []*ent.ListingOffer{row})
	if err != nil {
		return nil, false, fault.Wrap(err, fctx.With(ctx))
	}

	return offers[0], true, nil
}

func (r *Repository) ListOffers(ctx context.Context, id listing.ID) ([]*listing.Offer, error) {
	rows, err := r.offerQuery().
		Where(listingoffer.ListingID(xid.ID(id))).
		Order(listingoffer.ByUpdatedAt(sql.OrderDesc())).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return r.mapOffers(ctx, rows)
}

func (r *Repository) OfferedBy(ctx context.Context, buyer account.AccountID, ids ...listing.ID) (map[listing.ID]bool, error) {
	rows, err := r.db.ListingOffer.Query().
		Where(
			listingoffer.AccountID(xid.ID(buyer)),
			listingoffer.ListingIDIn(toXIDs(ids)...),
		).
		Select(listingoffer.FieldListingID).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make(map[listing.ID]bool, len(rows))
	for _, row := range rows {
		out[listing.ID(row.ListingID)] = true
	}
	return out, nil
}

func (r *Repository) OfferCounts(ctx context.Context, ids ...listing.ID) (map[listing.ID]int, error) {
	var rows []struct {
		ListingID xid.ID `json:"listing_id"`
		Count     int    `json:"count"`
	}

	err := r.db.ListingOffer.Query().
		Where(listingoffer.ListingIDIn(toXIDs(ids)...)).
		GroupBy(listingoffer.FieldListingID).
		Aggregate(ent.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make(map[listing.ID]int, len(rows))
	for _, row := range rows {
		out[listing.ID(row.ListingID)] = row.Count
	}
	return out, nil
}

func (r *Repository) AddFavourite(ctx context.Context, id listing.ID, acc account.AccountID) error {
	err := r.db.ListingFavourite.Create().
		SetListingID(xid.ID(id)).
		SetAccountID(xid.ID(acc)).
		OnConflictColumns(listingfavourite.FieldListingID, listingfavourite.FieldAccountID).
		DoNothing().
		Exec(ctx)
	if err != nil && !errors.Is(err, stdsql.ErrNoRows) {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (r *Repository) RemoveFavourite(ctx context.Context, id listing.ID, acc account.AccountID) error {
	_, err := r.db.ListingFavourite.Delete().
		Where(listingfavourite.ListingID(xid.ID(id)), listingfavourite.AccountID(xid.ID(acc))).
		Exec(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	return nil
}

func (r *Repository) FavouritedBy(ctx context.Context, acc account.AccountID, ids ...listing.ID) (map[listing.ID]bool, error) {
	rows, err := r.db.ListingFavourite.Query().
		Where(
			listingfavourite.AccountID(xid.ID(acc)),
			listingfavourite.ListingIDIn(toXIDs(ids)...),
		).
		Select(listingfavourite.FieldListingID).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := make(map[listing.ID]bool, len(rows))
	for _, row := range rows {
		out[listing.ID(row.ListingID)] = true
	}
	return out, nil
}

func toXIDs(ids []listing.ID) []xid.ID {
	return dt.Map(ids, func(id listing.ID) xid.ID { return xid.ID(id) })
}
