package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/rs/xid"
)

type Invitation struct {
	ent.Schema
}

func (Invitation) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}, DeletedAt{}}
}

func (Invitation) Fields() []ent.Field {
	return []ent.Field{
		field.String("message").
			Optional().
			Nillable(),

		field.String("creator_account_id").
			GoType(xid.ID{}),

		// MaxUses is the number of accounts that may register using this
		// invitation. Nil means unlimited uses.
		field.Int("max_uses").
			Optional().
			Nillable(),

		// ExpiresAt is the point in time after which this invitation may no
		// longer be used to register. Nil means it never expires.
		field.Time("expires_at").
			Optional().
			Nillable(),
	}
}

func (Invitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("creator", Account.Type).
			Ref("invitations").
			Field("creator_account_id").
			Unique().
			Required(),

		edge.To("invited", Account.Type),
	}
}
