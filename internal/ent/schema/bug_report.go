package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/rs/xid"
)

type BugReport struct {
	ent.Schema
}

func (BugReport) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}}
}

func (BugReport) Fields() []ent.Field {
	return []ent.Field{
		field.String("account_id").GoType(xid.ID{}),
		field.String("category"),
		field.Text("description"),
		field.Text("page_url").Optional().Nillable(),
		field.Text("user_agent").Optional().Nillable(),
		field.String("status").Default("open"),
		field.String("asset_id").GoType(xid.ID{}).Optional().Nillable(),
	}
}

func (BugReport) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("status"),
	}
}

func (BugReport) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("author", Account.Type).
			Field("account_id").
			Ref("bug_reports").
			Unique().
			Required(),

		edge.From("screenshot", Asset.Type).
			Field("asset_id").
			Ref("bug_reports").
			Unique(),
	}
}
