package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
	"github.com/rs/xid"
)

type DriveFolder struct {
	ent.Schema
}

func (DriveFolder) Mixin() []ent.Mixin {
	return []ent.Mixin{Identifier{}, CreatedAt{}, UpdatedAt{}, DeletedAt{}}
}

func (DriveFolder) Fields() []ent.Field {
	return []ent.Field{
		field.String("name").NotEmpty(),
		field.String("description").Optional(),
		field.String("drive_folder_id").NotEmpty(),
		field.Enum("visibility").Values("public", "member", "admin").Default("member"),
		field.Int("sort").Default(0),
		field.String("added_by").GoType(xid.ID{}),
	}
}

func (DriveFolder) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("account", Account.Type).
			Ref("drive_folders").
			Field("added_by").
			Required().
			Unique(),
	}
}

func (DriveFolder) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("drive_folder_id"),
	}
}
