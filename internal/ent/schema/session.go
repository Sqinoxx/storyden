package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"github.com/rs/xid"
)

type Session struct {
	ent.Schema
}

func (Session) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}}
}

func (Session) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_id").
			Immutable().
			GoType(xid.ID{}).
			NotEmpty(),

		// Mutable so that the sliding inactivity window can be extended.
		field.Time("expires_at"),

		field.Time("revoked_at").
			Optional().
			Nillable(),

		// Throttle anchor for the sliding window, so extending a session costs
		// at most one write per refresh interval rather than one per request.
		field.Time("refreshed_at").
			Optional().
			Nillable(),

		// The sliding window size captured when the session was issued. Zero
		// marks a session issued before configurable lifetimes existed, which
		// keeps its original expiry rather than being downgraded.
		field.Int("lifetime_seconds").
			Default(0),

		// Whether the cookie was issued as a persistent one, so refreshes
		// re-issue the same flavour.
		field.Bool("persistent").
			Default(false),
	}
}

func (Session) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("sessions").
			Field("account_id").
			Required().
			Immutable().
			Unique(),
	}
}
