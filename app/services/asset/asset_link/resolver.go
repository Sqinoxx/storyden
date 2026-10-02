// Package asset_link resolves asset references extracted from rich text
// content against the database, filtering out anything that no longer
// exists. This is required because IDs parsed out of user-supplied HTML may
// point at deleted or otherwise nonexistent assets, and blindly attaching
// them (e.g. via AddAssetIDs) would fail the whole write with a foreign key
// violation.
package asset_link

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_ref"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
)

type Resolver struct {
	db *ent.Client
}

func New(db *ent.Client) *Resolver {
	return &Resolver{db: db}
}

// Resolve extracts asset references from the given content and returns the
// subset that actually exist, preserving document order.
func (r *Resolver) Resolve(ctx context.Context, c datagraph.Content) ([]asset.AssetID, error) {
	return r.Existing(ctx, asset_ref.ExtractAssetIDs(c))
}

// Existing returns the unique, non-nil subset of ids that exist, preserving
// order. Explicit asset_ids from clients are as untrusted as IDs parsed out of
// HTML: an unparseable identifier becomes a nil ID and fails the edge insert.
func (r *Resolver) Existing(ctx context.Context, ids []asset.AssetID) ([]asset.AssetID, error) {
	seen := make(map[asset.AssetID]bool, len(ids))
	candidates := make([]asset.AssetID, 0, len(ids))
	for _, id := range ids {
		if xid.ID(id).IsNil() || seen[id] {
			continue
		}
		seen[id] = true
		candidates = append(candidates, id)
	}

	if len(candidates) == 0 {
		return []asset.AssetID{}, nil
	}

	existing, err := r.db.Asset.Query().
		Where(ent_asset.IDIn(candidates...)).
		IDs(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	exists := make(map[asset.AssetID]bool, len(existing))
	for _, id := range existing {
		exists[id] = true
	}

	result := make([]asset.AssetID, 0, len(candidates))
	for _, id := range candidates {
		if exists[id] {
			result = append(result, id)
		}
	}

	return result, nil
}
