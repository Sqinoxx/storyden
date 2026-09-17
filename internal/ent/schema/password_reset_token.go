package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/rs/xid"
)

// PasswordResetToken records the jti of every password-reset token that has
// been issued, so that a token can be marked used and rejected on a second
// attempt even though the token itself is a stateless, self-verifying JWT.
type PasswordResetToken struct {
	ent.Schema
}

func (PasswordResetToken) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}}
}

func (PasswordResetToken) Fields() []ent.Field {
	return []ent.Field{
		field.String("jti").NotEmpty().Immutable(),
		field.String("account_id").GoType(xid.ID{}).Immutable(),
		field.Time("expires_at").Immutable(),
		field.Time("used_at").Optional().Nillable(),
	}
}

func (PasswordResetToken) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("password_reset_tokens").
			Field("account_id").
			Required().
			Immutable().
			Unique(),
	}
}

func (PasswordResetToken) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("jti").Unique(),
	}
}
