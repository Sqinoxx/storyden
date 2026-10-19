package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/rs/xid"
)

type Listing struct {
	ent.Schema
}

func (Listing) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}, DeletedAt{}, ExpiresAt{}}
}

func (Listing) Fields() []ent.Field {
	return []ent.Field{
		field.String("title"),
		field.String("slug").Unique(),
		field.Text("description"),
		field.Int("price_cents").Default(0),
		field.Enum("price_type").Values("fixed", "negotiable", "free").Default("negotiable"),
		field.Enum("category").Values("books", "instruments", "electronics", "clothing", "furniture", "tickets", "other").Default("other"),
		field.Enum("condition").Values("new", "like_new", "used", "defective").Default("used"),
		field.String("location").Optional().Nillable(),
		field.Enum("status").Values("active", "reserved", "sold").Default("active"),
		field.JSON("asset_order", []string{}).Optional(),
		field.Time("expiry_notified_at").Optional().Nillable(),
		field.String("account_id").GoType(xid.ID{}),
	}
}

func (Listing) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("slug").Unique(),
		index.Fields("expires_at"),
		index.Fields("category"),
	}
}

func (Listing) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", Account.Type).
			Ref("listings").
			Field("account_id").
			Unique().
			Required(),

		edge.To("assets", Asset.Type),

		edge.To("offers", ListingOffer.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),

		edge.To("favourites", ListingFavourite.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

type ListingOffer struct {
	ent.Schema
}

func (ListingOffer) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}}
}

func (ListingOffer) Fields() []ent.Field {
	return []ent.Field{
		field.Int("amount_cents"),
		field.Text("message").Optional(),
		field.String("listing_id").GoType(xid.ID{}),
		field.String("account_id").GoType(xid.ID{}),
	}
}

func (ListingOffer) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("listing", Listing.Type).
			Ref("offers").
			Field("listing_id").
			Unique().
			Required(),

		edge.From("buyer", Account.Type).
			Ref("listing_offers").
			Field("account_id").
			Unique().
			Required(),
	}
}

func (ListingOffer) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("listing_id", "account_id").
			Unique().
			StorageKey("unique_listing_offer"),
	}
}

type ListingFavourite struct {
	ent.Schema
}

func (ListingFavourite) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}}
}

func (ListingFavourite) Fields() []ent.Field {
	return []ent.Field{
		field.String("listing_id").GoType(xid.ID{}),
		field.String("account_id").GoType(xid.ID{}),
	}
}

func (ListingFavourite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("listing", Listing.Type).
			Ref("favourites").
			Field("listing_id").
			Unique().
			Required(),

		edge.From("account", Account.Type).
			Ref("listing_favourites").
			Field("account_id").
			Unique().
			Required(),
	}
}

func (ListingFavourite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("listing_id", "account_id").
			Unique().
			StorageKey("unique_listing_favourite"),
	}
}
