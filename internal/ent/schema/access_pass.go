package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/dialect/entsql"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
)

type AccessPass struct {
	ent.Schema
}

func (AccessPass) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}}
}

func (AccessPass) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty().MaxLen(200),
		field.String("description").Default(""),
		field.String("price_label").Default(""),
		field.String("payment_info").Default(""),
		field.String("colour").Default("#8577ce"),
		field.Int("sort").Default(0),
		field.Bool("requestable").Default(true),
	}
}

func (AccessPass) Edges() []ent.Edge {
	return []ent.Edge{
		edge.To("categories", Category.Type),
		edge.To("drive_folders", DriveFolder.Type),
		edge.To("grants", AccessGrant.Type).
			Annotations(entsql.OnDelete(entsql.Cascade)),
	}
}
