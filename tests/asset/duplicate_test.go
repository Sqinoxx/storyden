package asset_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/services/asset/asset_hash"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/ent"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestAsset_DuplicateDetection(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		db *ent.Client,
		backfill *asset_hash.Backfill,
	) {
		lc.Append(fx.StartHook(func() {
			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name:        "dup-" + xid.New().String(),
				Description: "d",
				Colour:      "#fff",
			}, session))(t, http.StatusOK).JSON200.Id

			createThread := func(vis openapi.Visibility, title string, assetID string) string {
				ids := []string{assetID}
				return tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      title,
					Category:   &cat,
					Body:       opt.New("<p>body</p>").Ptr(),
					Visibility: opt.New(vis).Ptr(),
					AssetIds:   &ids,
				}, session))(t, http.StatusOK).JSON200.Slug
			}

			content := encodePNG(3, 5)

			t.Run("first_upload_is_not_a_duplicate", func(t *testing.T) {
				a := uploadNamedAsset(t, root, cl, session, "image/png", "original.png", content)
				require.Nil(t, a.DuplicateOf)
			})

			t.Run("same_content_different_name_is_detected", func(t *testing.T) {
				r := require.New(t)
				a := uploadNamedAsset(t, root, cl, session, "image/png", "renamed.png", encodePNG(4, 6))
				r.Nil(a.DuplicateOf)

				published := createThread(openapi.VisibilityPublished, "Published thread", a.Id)
				createThread(openapi.VisibilityDraft, "Secret draft", uploadNamedAsset(t, root, cl, session, "image/png", "x.png", encodePNG(4, 6)).Id)

				dup := uploadNamedAsset(t, root, cl, session, "image/png", "totally-different-name.png", encodePNG(4, 6))
				r.NotNil(dup.DuplicateOf)
				r.Equal(2, dup.DuplicateOf.Count)
				r.Len(dup.DuplicateOf.Locations, 1, "the draft must not be revealed")
				r.Equal(openapi.AssetDuplicateLocationKindThread, dup.DuplicateOf.Locations[0].Kind)
				r.Equal(published, dup.DuplicateOf.Locations[0].Slug)
				r.Equal("Published thread", dup.DuplicateOf.Locations[0].Title)
			})

			t.Run("same_name_different_content_is_not_detected", func(t *testing.T) {
				uploadNamedAsset(t, root, cl, session, "image/png", "same.png", encodePNG(7, 7))
				a := uploadNamedAsset(t, root, cl, session, "image/png", "same.png", encodePNG(8, 8))
				require.Nil(t, a.DuplicateOf)
			})

			t.Run("backfill_hashes_legacy_assets", func(t *testing.T) {
				r := require.New(t)
				legacy := uploadNamedAsset(t, root, cl, session, "image/png", "legacy.png", encodePNG(9, 2))
				r.NoError(db.Asset.UpdateOneID(openapi.ParseID(legacy.Id)).ClearContentHash().Exec(root))

				_, err := backfill.Run(root)
				r.NoError(err)

				row, err := db.Asset.Query().Where(ent_asset.ID(openapi.ParseID(legacy.Id))).Only(root)
				r.NoError(err)
				r.NotNil(row.ContentHash)
				r.NotEmpty(*row.ContentHash)

				dup := uploadNamedAsset(t, root, cl, session, "image/png", "new.png", encodePNG(9, 2))
				r.NotNil(dup.DuplicateOf)
				r.Equal(1, dup.DuplicateOf.Count)
			})
		}))
	}))
}
