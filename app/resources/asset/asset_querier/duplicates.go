package asset_querier

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

const maxDuplicateLocations = 5

// FindDuplicates only reports locations that are published, so an identical
// file sitting in someone's draft or a deleted thread is counted but never
// named.
func (q *Querier) FindDuplicates(ctx context.Context, hash string, exclude asset.AssetID) (*asset.DuplicateInfo, error) {
	if hash == "" {
		return nil, nil
	}

	ids, err := q.db.Asset.Query().
		Where(
			ent_asset.ContentHash(hash),
			ent_asset.IDNEQ(xid.ID(exclude)),
		).
		IDs(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if len(ids) == 0 {
		return nil, nil
	}

	posts, err := q.db.Post.Query().
		Where(
			ent_post.HasAssetsWith(ent_asset.IDIn(ids...)),
			ent_post.DeletedAtIsNil(),
			ent_post.VisibilityEQ(ent_post.VisibilityPublished),
		).
		Select(ent_post.FieldID, ent_post.FieldRootPostID).
		All(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	threadIDs := make([]xid.ID, 0, len(posts))
	for _, p := range posts {
		if p.RootPostID != nil {
			threadIDs = append(threadIDs, *p.RootPostID)
		} else {
			threadIDs = append(threadIDs, p.ID)
		}
	}

	locations := []asset.DuplicateLocation{}

	if len(threadIDs) > 0 {
		threads, err := q.db.Post.Query().
			Where(
				ent_post.IDIn(threadIDs...),
				ent_post.RootPostIDIsNil(),
				ent_post.DeletedAtIsNil(),
				ent_post.VisibilityEQ(ent_post.VisibilityPublished),
			).
			Select(ent_post.FieldID, ent_post.FieldSlug, ent_post.FieldTitle).
			Order(ent_post.ByID()).
			Limit(maxDuplicateLocations).
			All(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		for _, t := range threads {
			locations = append(locations, asset.DuplicateLocation{
				Kind:  asset.DuplicateLocationThread,
				Slug:  t.Slug,
				Title: t.Title,
			})
		}
	}

	if remaining := maxDuplicateLocations - len(locations); remaining > 0 {
		nodes, err := q.db.Node.Query().
			Where(
				ent_node.Or(
					ent_node.HasAssetsWith(ent_asset.IDIn(ids...)),
					ent_node.PrimaryAssetIDIn(ids...),
				),
				ent_node.DeletedAtIsNil(),
				ent_node.VisibilityEQ(ent_node.VisibilityPublished),
			).
			Select(ent_node.FieldID, ent_node.FieldSlug, ent_node.FieldName).
			Order(ent_node.ByID()).
			Limit(remaining).
			All(ctx)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
		for _, n := range nodes {
			locations = append(locations, asset.DuplicateLocation{
				Kind:  asset.DuplicateLocationNode,
				Slug:  n.Slug,
				Title: n.Name,
			})
		}
	}

	return &asset.DuplicateInfo{
		Count:     len(ids),
		Locations: locations,
	}, nil
}
