package library

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_node "github.com/Southclaws/storyden/internal/ent/node"
)

func TestItemRefIncludesOwnerAndAssetsForIndexing(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	ocrText := "node ocr"
	ownerID := xid.New()
	content := "<p>node body</p>"

	item, err := ItemRef(&ent.Node{
		ID:         xid.New(),
		AccountID:  ownerID,
		Name:       "node",
		Slug:       "node",
		Content:    &content,
		Visibility: ent_node.VisibilityPublished,
		Edges: ent.NodeEdges{
			Assets: []*ent.Asset{
				{
					ID:        xid.New(),
					Filename:  "asset.png",
					MimeType:  "image/png",
					OcrStatus: ent_asset.OcrStatusCompleted,
					OcrText:   &ocrText,
				},
			},
		},
	})
	r.NoError(err)

	nodeItem, ok := item.(*Node)
	r.True(ok)
	a.Equal(ownerID, xid.ID(nodeItem.Owner.ID))
	r.Len(nodeItem.Assets, 1)
	text, ok := nodeItem.Assets[0].OCRText.Get()
	r.True(ok)
	a.Equal(ocrText, text)
}
