package reply

import (
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

func TestItemRefIncludesAuthorAndAssetsForIndexing(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	ocrText := "reply ocr"
	ownerID := xid.New()
	rootID := xid.New()

	item, err := ItemRef(&ent.Post{
		ID:           xid.New(),
		AccountPosts: ownerID,
		RootPostID:   &rootID,
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

	replyItem, ok := item.(*Reply)
	r.True(ok)
	a.Equal(ownerID, xid.ID(replyItem.Author.ID))
	r.Len(replyItem.Assets, 1)
	text, ok := replyItem.Assets[0].OCRText.Get()
	r.True(ok)
	a.Equal(ocrText, text)
}
