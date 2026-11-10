package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/rs/xid"
)

type AccessGrant struct {
	ent.Schema
}

func (AccessGrant) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}, ExpiresAt{}}
}

func (AccessGrant) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_id").GoType(xid.ID{}),
		field.String("pass_id").GoType(xid.ID{}),
		field.Enum("status").Values("requested", "active", "rejected", "revoked"),
		field.String("reference_code").Unique().NotEmpty(),
		field.Time("requested_at").Optional().Nillable(),
		field.String("decided_by").GoType(xid.ID{}).Optional().Nillable(),
		field.Time("decided_at").Optional().Nillable(),
		field.String("note").Default(""),
		field.Time("expiry_notified_at").Optional().Nillable(),
	}
}

func (AccessGrant) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Field("account_id").
			Ref("access_grants").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
		edge.From("pass", AccessPass.Type).
			Field("pass_id").
			Ref("grants").
			Unique().
			Required().
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}

func (AccessGrant) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("account_id", "pass_id").Unique(),
		index.Fields("pass_id", "status"),
	}
}
