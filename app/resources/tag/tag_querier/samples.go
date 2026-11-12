package tag_querier

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/tag/tag_ref"
	"github.com/Southclaws/storyden/internal/ent"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	ent_tag "github.com/Southclaws/storyden/internal/ent/tag"
)

type ThreadSample struct {
	Title string
	Short string
}

func (q *Querier) SampleThreads(ctx context.Context, name tag_ref.Name, limit int) ([]ThreadSample, error) {
	r, err := q.db.Post.Query().
		Where(
			ent_post.HasTagsWith(ent_tag.Name(name.String())),
			ent_post.RootPostIDIsNil(),
			ent_post.VisibilityEQ(ent_post.VisibilityPublished),
			ent_post.DeletedAtIsNil(),
		).
		Order(ent.Desc(ent_post.FieldCreatedAt)).
		Limit(limit).
		Select(ent_post.FieldTitle, ent_post.FieldShort).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.Map(r, func(p *ent.Post) ThreadSample {
		return ThreadSample{Title: p.Title, Short: p.Short}
	}), nil
}
