package asset_querier

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

func (q *Querier) ListExtractedTextForPosts(ctx context.Context, postIDs []xid.ID) ([]*asset.Asset, error) {
	if len(postIDs) == 0 {
		return nil, nil
	}

	r, err := q.db.Asset.Query().
		Where(
			ent_asset.HasPostsWith(ent_post.IDIn(postIDs...)),
			ent_asset.OcrStatusEQ(ent_asset.OcrStatusCompleted),
			ent_asset.OcrTextNotNil(),
			ent_asset.OcrTextNEQ(""),
		).
		Order(ent.Asc(ent_asset.FieldID)).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return dt.Map(r, asset.Map), nil
}
