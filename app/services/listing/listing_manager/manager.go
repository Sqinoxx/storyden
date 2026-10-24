package listing_manager

import (
	"context"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_phone"
	"github.com/Southclaws/storyden/app/resources/account/account_querier"
	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/listing"
	"github.com/Southclaws/storyden/app/resources/listing/listing_repo"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/profile"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/notification/notify"
)

const RenewWindow = 7 * 24 * time.Hour

var (
	ErrNotAuthorised = fault.New("not authorised", ftag.With(ftag.PermissionDenied))
	ErrPhoneRequired = fault.New("phone number required", ftag.With(ftag.InvalidArgument))
	ErrOwnListing    = fault.New("cannot make an offer on own listing", ftag.With(ftag.InvalidArgument))
	ErrUnavailable   = fault.New("listing is no longer available", ftag.With(ftag.InvalidArgument))
	ErrNoOffer       = fault.New("no offer made", ftag.With(ftag.PermissionDenied))
	ErrTooEarly      = fault.New("listing cannot be renewed yet", ftag.With(ftag.InvalidArgument))
	ErrTooManyAssets = fault.New("too many images", ftag.With(ftag.InvalidArgument))
	ErrDisabled      = fault.New("flohmarkt is not enabled", ftag.With(ftag.NotFound))
	ErrBadCategory   = fault.New("unknown category", ftag.With(ftag.InvalidArgument))
)

type Manager struct {
	accountQuery *account_querier.Querier
	phones       *account_phone.Repository
	repo         *listing_repo.Repository
	settings     *settings.SettingsRepository
	notifier     *notify.Notifier
}

func New(
	accountQuery *account_querier.Querier,
	phones *account_phone.Repository,
	repo *listing_repo.Repository,
	settings *settings.SettingsRepository,
	notifier *notify.Notifier,
) *Manager {
	return &Manager{
		accountQuery: accountQuery,
		phones:       phones,
		repo:         repo,
		settings:     settings,
		notifier:     notifier,
	}
}

type View struct {
	*listing.Listing
	IsOwn       bool
	CanManage   bool
	IsExpired   bool
	HasOffered  bool
	IsFavourite bool
	OfferCount  opt.Optional[int]
}

type Partial struct {
	Title       opt.Optional[string]
	Description opt.Optional[string]
	PriceCents  opt.Optional[int]
	PriceType   opt.Optional[listing.PriceType]
	Category    opt.Optional[string]
	Condition   opt.Optional[listing.Condition]
	Status      opt.Optional[listing.Status]
	Location    opt.Optional[string]
	Assets      opt.Optional[[]asset.AssetID]
}

func (p Partial) Opts() []listing_repo.Option {
	opts := []listing_repo.Option{}
	p.Title.Call(func(v string) { opts = append(opts, listing_repo.WithTitle(v)) })
	p.Description.Call(func(v string) { opts = append(opts, listing_repo.WithDescription(v)) })
	p.PriceCents.Call(func(v int) { opts = append(opts, listing_repo.WithPrice(v)) })
	p.PriceType.Call(func(v listing.PriceType) { opts = append(opts, listing_repo.WithPriceType(v)) })
	p.Category.Call(func(v string) { opts = append(opts, listing_repo.WithCategory(v)) })
	p.Condition.Call(func(v listing.Condition) { opts = append(opts, listing_repo.WithCondition(v)) })
	p.Status.Call(func(v listing.Status) { opts = append(opts, listing_repo.WithStatus(v)) })
	p.Location.Call(func(v string) { opts = append(opts, listing_repo.WithLocation(opt.New(v))) })
	p.Assets.Call(func(v []asset.AssetID) { opts = append(opts, listing_repo.WithAssets(v)) })
	return opts
}

func (m *Manager) viewer(ctx context.Context) (*account.AccountWithEdges, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	acc, err := m.accountQuery.GetByID(ctx, accountID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	s, err := m.settings.Get(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if !s.MarketEnabled() && !isModerator(acc) {
		return nil, fault.Wrap(ErrDisabled, fctx.With(ctx))
	}

	return acc, nil
}

func isModerator(acc *account.AccountWithEdges) bool {
	return acc.Roles.Permissions().HasAny(rbac.PermissionManageListings, rbac.PermissionAdministrator)
}

func (m *Manager) authoriseOwner(ctx context.Context, acc *account.AccountWithEdges, l *listing.Listing) error {
	return acc.Roles.Permissions().Authorise(ctx, func() error {
		if l.Owner.ID != acc.ID {
			return fault.Wrap(ErrNotAuthorised, fctx.With(ctx),
				fmsg.WithDesc("not owner", "Nur der Verkäufer darf dieses Inserat bearbeiten."))
		}
		return nil
	}, rbac.PermissionManageListings)
}

func (m *Manager) requirePhone(ctx context.Context, id account.AccountID) error {
	phone, err := m.phones.Get(ctx, id)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	if !phone.Ok() {
		return fault.Wrap(ErrPhoneRequired, fctx.With(ctx),
			fmsg.WithDesc("phone required", "Bitte hinterlege zuerst eine Telefonnummer in deinen Kontoeinstellungen."))
	}
	return nil
}

func (m *Manager) validateCategory(ctx context.Context, category opt.Optional[string]) error {
	c, ok := category.Get()
	if !ok {
		return nil
	}

	s, err := m.settings.Get(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if !s.HasMarketCategory(c) {
		return fault.Wrap(ErrBadCategory, fctx.With(ctx),
			fmsg.WithDesc("unknown category", "Diese Kategorie gibt es nicht."))
	}
	return nil
}

func (m *Manager) duration(ctx context.Context) (time.Duration, error) {
	s, err := m.settings.Get(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}
	return s.ListingDuration(), nil
}

func validateAssets(ctx context.Context, assets opt.Optional[[]asset.AssetID]) error {
	if a, ok := assets.Get(); ok && len(a) > listing.MaxAssets {
		return fault.Wrap(ErrTooManyAssets, fctx.With(ctx),
			fmsg.WithDesc("too many assets", "Es sind maximal 10 Bilder pro Inserat erlaubt."))
	}
	return nil
}

func (m *Manager) Create(ctx context.Context, title string, partial Partial) (*View, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := acc.RejectSuspended(); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.requirePhone(ctx, acc.ID); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := validateAssets(ctx, partial.Assets); err != nil {
		return nil, err
	}

	if err := m.validateCategory(ctx, partial.Category); err != nil {
		return nil, err
	}

	d, err := m.duration(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	partial.Status = opt.NewEmpty[listing.Status]()

	l, err := m.repo.Create(ctx, acc.ID, title, time.Now().Add(d), partial.Opts()...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return m.annotate(ctx, acc, l)
}

func (m *Manager) Get(ctx context.Context, key listing.QueryKey) (*View, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	v, err := m.annotate(ctx, acc, l)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if v.IsExpired && !v.IsOwn && !v.HasOffered && !isModerator(acc) {
		return nil, fault.New("listing expired", fctx.With(ctx), ftag.With(ftag.NotFound))
	}

	return v, nil
}

type Scope string

const (
	ScopeAll        Scope = "all"
	ScopeMine       Scope = "mine"
	ScopeFavourites Scope = "favourites"
	ScopeManage     Scope = "manage"
)

type ListParams struct {
	Page      pagination.Parameters
	Sort      listing_repo.Sort
	Scope     Scope
	Query     opt.Optional[string]
	Category  opt.Optional[string]
	Condition opt.Optional[listing.Condition]
	PriceType opt.Optional[listing.PriceType]
	PriceMin  opt.Optional[int]
	PriceMax  opt.Optional[int]
}

type ListResult struct {
	*pagination.Result[*View]
	PriceMin int
	PriceMax int
}

func (m *Manager) List(ctx context.Context, p ListParams) (*ListResult, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	filters := []listing_repo.Filter{}

	switch p.Scope {
	case ScopeMine:
		filters = append(filters, listing_repo.WithOwner(acc.ID))
	case ScopeFavourites:
		filters = append(filters, listing_repo.WithFavouritedBy(acc.ID), listing_repo.WithActiveAt(time.Now()))
	case ScopeManage:
		if !isModerator(acc) {
			return nil, fault.Wrap(ErrNotAuthorised, fctx.With(ctx),
				fmsg.WithDesc("not moderator", "Nur Moderatoren dürfen alle Inserate verwalten."))
		}
	default:
		filters = append(filters, listing_repo.WithActiveAt(time.Now()))
	}

	p.Query.Call(func(v string) {
		if v != "" {
			filters = append(filters, listing_repo.WithQuery(v))
		}
	})
	p.Category.Call(func(v string) { filters = append(filters, listing_repo.WithCategories(v)) })
	p.Condition.Call(func(v listing.Condition) { filters = append(filters, listing_repo.WithConditions(v)) })
	p.PriceType.Call(func(v listing.PriceType) { filters = append(filters, listing_repo.WithPriceTypes(v)) })
	priceMin, priceMax, err := m.repo.PriceBounds(ctx, filters...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	p.PriceMin.Call(func(v int) { filters = append(filters, listing_repo.WithPriceMin(v)) })
	p.PriceMax.Call(func(v int) { filters = append(filters, listing_repo.WithPriceMax(v)) })

	result, err := m.repo.List(ctx, p.Page, p.Sort, filters...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	views, err := m.annotateMany(ctx, acc, result.Items)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := pagination.ConvertPageResult(*result, views)

	return &ListResult{Result: &out, PriceMin: priceMin, PriceMax: priceMax}, nil
}

func (m *Manager) Update(ctx context.Context, key listing.QueryKey, partial Partial) (*View, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if l.Owner.ID != acc.ID {
		return nil, fault.Wrap(ErrNotAuthorised, fctx.With(ctx),
			fmsg.WithDesc("not owner", "Nur der Verkäufer darf dieses Inserat bearbeiten."))
	}

	if err := validateAssets(ctx, partial.Assets); err != nil {
		return nil, err
	}

	if c, ok := partial.Category.Get(); ok && c != l.Category {
		if err := m.validateCategory(ctx, partial.Category); err != nil {
			return nil, err
		}
	}

	l, err = m.repo.Update(ctx, l.ID, partial.Opts()...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return m.annotate(ctx, acc, l)
}

func (m *Manager) Delete(ctx context.Context, key listing.QueryKey) error {
	acc, err := m.viewer(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.authoriseOwner(ctx, acc, l); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return m.repo.Delete(ctx, l.ID)
}

func (m *Manager) Renew(ctx context.Context, key listing.QueryKey) (*View, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.authoriseOwner(ctx, acc, l); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	now := time.Now()
	if l.ExpiresAt.After(now.Add(RenewWindow)) {
		return nil, fault.Wrap(ErrTooEarly, fctx.With(ctx),
			fmsg.WithDesc("too early", "Ein Inserat kann frühestens 7 Tage vor Ablauf verlängert werden."))
	}

	d, err := m.duration(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err = m.repo.Update(ctx, l.ID, listing_repo.WithExpiresAt(now.Add(d)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return m.annotate(ctx, acc, l)
}

func (m *Manager) MakeOffer(ctx context.Context, key listing.QueryKey, amountCents int, message string) (*listing.Offer, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := acc.RejectSuspended(); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if l.Owner.ID == acc.ID {
		return nil, fault.Wrap(ErrOwnListing, fctx.With(ctx),
			fmsg.WithDesc("own listing", "Du kannst kein Angebot für dein eigenes Inserat machen."))
	}

	if l.IsExpired(time.Now()) || l.Status == listing.StatusSold {
		return nil, fault.Wrap(ErrUnavailable, fctx.With(ctx),
			fmsg.WithDesc("unavailable", "Dieses Inserat ist nicht mehr verfügbar."))
	}

	if err := m.requirePhone(ctx, acc.ID); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	_, existed, err := m.repo.GetOffer(ctx, l.ID, acc.ID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	offer, err := m.repo.UpsertOffer(ctx, l.ID, acc.ID, amountCents, message)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if !existed {
		if err := m.notifier.Send(ctx, l.Owner.ID, opt.New(acc.ID), notification.EventListingOfferReceived, nil); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
	}

	return offer, nil
}

type Contact struct {
	Phone  string
	Seller profile.Ref
}

func (m *Manager) Contact(ctx context.Context, key listing.QueryKey) (*Contact, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if l.Owner.ID != acc.ID {
		_, ok, err := m.repo.GetOffer(ctx, l.ID, acc.ID)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		if !ok {
			return nil, fault.Wrap(ErrNoOffer, fctx.With(ctx),
				fmsg.WithDesc("no offer", "Mach zuerst ein Angebot, um die Telefonnummer zu sehen."))
		}
	}

	phone, err := m.phones.Get(ctx, l.Owner.ID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Contact{
		Phone:  phone.OrZero(),
		Seller: l.Owner,
	}, nil
}

type OfferWithPhone struct {
	*listing.Offer
	Phone opt.Optional[string]
}

func (m *Manager) Offers(ctx context.Context, key listing.QueryKey) ([]*OfferWithPhone, error) {
	acc, err := m.viewer(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := m.authoriseOwner(ctx, acc, l); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	offers, err := m.repo.ListOffers(ctx, l.ID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	phones, err := m.phones.GetMany(ctx, dt.Map(offers, func(o *listing.Offer) account.AccountID { return o.Buyer.ID })...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.Map(offers, func(o *listing.Offer) *OfferWithPhone {
		p, ok := phones[o.Buyer.ID]
		return &OfferWithPhone{Offer: o, Phone: opt.NewSafe(p, ok)}
	}), nil
}

func (m *Manager) SetFavourite(ctx context.Context, key listing.QueryKey, favourite bool) error {
	acc, err := m.viewer(ctx)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	l, err := m.repo.Get(ctx, key)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if favourite {
		return m.repo.AddFavourite(ctx, l.ID, acc.ID)
	}
	return m.repo.RemoveFavourite(ctx, l.ID, acc.ID)
}

func (m *Manager) annotate(ctx context.Context, acc *account.AccountWithEdges, l *listing.Listing) (*View, error) {
	views, err := m.annotateMany(ctx, acc, []*listing.Listing{l})
	if err != nil {
		return nil, err
	}
	return views[0], nil
}

func (m *Manager) annotateMany(ctx context.Context, acc *account.AccountWithEdges, ls []*listing.Listing) ([]*View, error) {
	ids := dt.Map(ls, func(l *listing.Listing) listing.ID { return l.ID })

	offered, err := m.repo.OfferedBy(ctx, acc.ID, ids...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	favourites, err := m.repo.FavouritedBy(ctx, acc.ID, ids...)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	mod := isModerator(acc)

	manageIDs := dt.Map(dt.Filter(ls, func(l *listing.Listing) bool { return mod || l.Owner.ID == acc.ID }),
		func(l *listing.Listing) listing.ID { return l.ID })

	counts := map[listing.ID]int{}
	if len(manageIDs) > 0 {
		counts, err = m.repo.OfferCounts(ctx, manageIDs...)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
	}

	now := time.Now()

	return dt.Map(ls, func(l *listing.Listing) *View {
		own := l.Owner.ID == acc.ID
		return &View{
			Listing:     l,
			IsOwn:       own,
			CanManage:   own || mod,
			IsExpired:   l.IsExpired(now),
			HasOffered:  offered[l.ID],
			IsFavourite: favourites[l.ID],
			OfferCount:  opt.NewSafe(counts[l.ID], own || mod),
		}
	}), nil
}
