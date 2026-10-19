package listing

import (
	"sort"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/mark"
	"github.com/Southclaws/storyden/app/resources/profile"
	"github.com/Southclaws/storyden/internal/ent"
	ent_listing "github.com/Southclaws/storyden/internal/ent/listing"
	"github.com/Southclaws/storyden/internal/ent/predicate"
)

//go:generate go run -mod=mod github.com/Southclaws/enumerator

type priceTypeEnum string

const (
	priceTypeFixed      priceTypeEnum = "fixed"
	priceTypeNegotiable priceTypeEnum = "negotiable"
	priceTypeFree       priceTypeEnum = "free"
)

type categoryEnum string

const (
	categoryBooks       categoryEnum = "books"
	categoryInstruments categoryEnum = "instruments"
	categoryElectronics categoryEnum = "electronics"
	categoryClothing    categoryEnum = "clothing"
	categoryFurniture   categoryEnum = "furniture"
	categoryTickets     categoryEnum = "tickets"
	categoryOther       categoryEnum = "other"
)

type conditionEnum string

const (
	conditionNew       conditionEnum = "new"
	conditionLikeNew   conditionEnum = "like_new"
	conditionUsed      conditionEnum = "used"
	conditionDefective conditionEnum = "defective"
)

type statusEnum string

const (
	statusActive   statusEnum = "active"
	statusReserved statusEnum = "reserved"
	statusSold     statusEnum = "sold"
)

const MaxAssets = 10

type ID xid.ID

func (i ID) String() string { return xid.ID(i).String() }

type (
	QueryKey struct{ mark.Queryable }
)

func NewKey(in string) QueryKey { return QueryKey{mark.NewQueryKey(in)} }
func NewID(id ID) QueryKey      { return QueryKey{mark.NewQueryKeyID(xid.ID(id))} }

func (m QueryKey) Predicate() (p predicate.Listing) {
	m.Apply(
		func(i xid.ID) { p = ent_listing.ID(i) },
		func(s string) { p = ent_listing.Slug(s) })
	return
}

type Listing struct {
	ID          ID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ExpiresAt   time.Time
	Title       string
	Slug        string
	Description string
	PriceCents  int
	PriceType   PriceType
	Category    Category
	Condition   Condition
	Status      Status
	Location    opt.Optional[string]
	Assets      []*asset.Asset
	Owner       profile.Ref
}

func (l *Listing) IsExpired(now time.Time) bool {
	return !l.ExpiresAt.After(now)
}

func (l *Listing) Mark() string {
	return l.Slug
}

type Offer struct {
	ID          xid.ID
	CreatedAt   time.Time
	UpdatedAt   time.Time
	ListingID   ID
	AmountCents int
	Message     string
	Buyer       profile.Ref
}

func Map(in *ent.Listing) (*Listing, error) {
	ownerEdge, err := in.Edges.OwnerOrErr()
	if err != nil {
		return nil, fault.Wrap(err)
	}

	owner, err := profile.MapRef(ownerEdge)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	priceType, err := NewPriceType(in.PriceType.String())
	if err != nil {
		return nil, fault.Wrap(err)
	}
	category, err := NewCategory(in.Category.String())
	if err != nil {
		return nil, fault.Wrap(err)
	}
	condition, err := NewCondition(in.Condition.String())
	if err != nil {
		return nil, fault.Wrap(err)
	}
	status, err := NewStatus(in.Status.String())
	if err != nil {
		return nil, fault.Wrap(err)
	}

	assets := dt.Map(in.Edges.Assets, asset.Map)
	sortAssets(assets, in.AssetOrder)

	var expires time.Time
	if in.ExpiresAt != nil {
		expires = *in.ExpiresAt
	}

	return &Listing{
		ID:          ID(in.ID),
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		ExpiresAt:   expires,
		Title:       in.Title,
		Slug:        in.Slug,
		Description: in.Description,
		PriceCents:  in.PriceCents,
		PriceType:   priceType,
		Category:    category,
		Condition:   condition,
		Status:      status,
		Location:    opt.NewPtr(in.Location),
		Assets:      assets,
		Owner:       *owner,
	}, nil
}

func sortAssets(assets []*asset.Asset, order []string) {
	position := make(map[string]int, len(order))
	for i, id := range order {
		position[id] = i
	}
	rank := func(a *asset.Asset) int {
		if i, ok := position[a.ID.String()]; ok {
			return i
		}
		return len(order)
	}
	sort.SliceStable(assets, func(i, j int) bool { return rank(assets[i]) < rank(assets[j]) })
}

func MapOffer(in *ent.ListingOffer) (*Offer, error) {
	buyerEdge, err := in.Edges.BuyerOrErr()
	if err != nil {
		return nil, fault.Wrap(err)
	}

	buyer, err := profile.MapRef(buyerEdge)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	return &Offer{
		ID:          in.ID,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		ListingID:   ID(in.ListingID),
		AmountCents: in.AmountCents,
		Message:     in.Message,
		Buyer:       *buyer,
	}, nil
}
