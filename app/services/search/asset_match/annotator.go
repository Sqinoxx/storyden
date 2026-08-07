// Package asset_match finds which attached assets (PDFs, images) matched a
// search query via their extracted text, and builds a short excerpt for
// display, independently of which search provider (database/bleve/redis)
// produced the results.
package asset_match

import (
	"context"
	"sort"
	"strings"

	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

const (
	excerptRadius = 90
	maxPerItem    = 3
)

type Match struct {
	Asset   *asset.Asset
	Excerpt string
}

type Annotator struct {
	db *ent.Client
}

func New(db *ent.Client) *Annotator {
	return &Annotator{db: db}
}

// Annotate returns, per item ID, the attached assets whose extracted text
// contains the query, each with a short excerpt around the first match.
// Items with no matching attachment are simply absent from the result map.
func (an *Annotator) Annotate(ctx context.Context, query string, items []datagraph.Item) (map[xid.ID][]Match, error) {
	query = strings.TrimSpace(query)
	if query == "" || len(items) == 0 {
		return nil, nil
	}

	var postIDs, nodeIDs []xid.ID
	for _, item := range items {
		switch item.GetKind() {
		case datagraph.KindPost, datagraph.KindThread, datagraph.KindReply:
			postIDs = append(postIDs, item.GetID())
		case datagraph.KindNode:
			nodeIDs = append(nodeIDs, item.GetID())
		}
	}

	result := make(map[xid.ID][]Match)

	if len(postIDs) > 0 {
		if err := an.annotatePosts(ctx, query, postIDs, result); err != nil {
			return nil, err
		}
	}

	if len(nodeIDs) > 0 {
		if err := an.annotateNodes(ctx, query, nodeIDs, result); err != nil {
			return nil, err
		}
	}

	return result, nil
}

func (an *Annotator) annotatePosts(ctx context.Context, query string, postIDs []xid.ID, result map[xid.ID][]Match) error {
	assets, err := an.db.Asset.Query().
		Where(
			ent_asset.OcrTextContainsFold(query),
			ent_asset.HasPostsWith(ent_post.IDIn(postIDs...)),
		).
		WithPosts(func(pq *ent.PostQuery) {
			pq.Where(ent_post.IDIn(postIDs...))
		}).
		All(ctx)
	if err != nil {
		return err
	}

	for _, a := range sortedByID(assets) {
		match := Match{
			Asset:   asset.Map(a),
			Excerpt: Excerpt(ocrText(a), query, excerptRadius),
		}
		for _, p := range a.Edges.Posts {
			if len(result[p.ID]) < maxPerItem {
				result[p.ID] = append(result[p.ID], match)
			}
		}
	}

	return nil
}

func (an *Annotator) annotateNodes(ctx context.Context, query string, nodeIDs []xid.ID, result map[xid.ID][]Match) error {
	assets, err := an.db.Asset.Query().
		Where(
			ent_asset.OcrTextContainsFold(query),
			ent_asset.HasNodesWith(ent_node.IDIn(nodeIDs...)),
		).
		WithNodes(func(nq *ent.NodeQuery) {
			nq.Where(ent_node.IDIn(nodeIDs...))
		}).
		All(ctx)
	if err != nil {
		return err
	}

	for _, a := range sortedByID(assets) {
		match := Match{
			Asset:   asset.Map(a),
			Excerpt: Excerpt(ocrText(a), query, excerptRadius),
		}
		for _, n := range a.Edges.Nodes {
			if len(result[n.ID]) < maxPerItem {
				result[n.ID] = append(result[n.ID], match)
			}
		}
	}

	return nil
}

func sortedByID(assets []*ent.Asset) []*ent.Asset {
	sort.Slice(assets, func(i, j int) bool {
		return assets[i].ID.String() < assets[j].ID.String()
	})
	return assets
}

func ocrText(a *ent.Asset) string {
	if a.OcrText == nil {
		return ""
	}
	return *a.OcrText
}
