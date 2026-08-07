package thread

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

func TestItemRefIncludesAssetsForIndexing(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	ocrText := "hello ocr"
	ownerID := xid.New()

	item, err := ItemRef(&ent.Post{
		ID:           xid.New(),
		AccountPosts: ownerID,
		Body:         "<p>body</p>",
		Visibility:   ent_post.VisibilityPublished,
		Edges: ent.PostEdges{
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

	threadItem, ok := item.(*Thread)
	r.True(ok)
	a.Equal(ownerID, xid.ID(threadItem.Author.ID))
	r.Len(threadItem.Assets, 1)
	text, ok := threadItem.Assets[0].OCRText.Get()
	r.True(ok)
	a.Equal(ocrText, text)
}
