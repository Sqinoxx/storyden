package thread_test

import (
	"bytes"
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestThreadAssetIDsIgnoreUnparseableIdentifiers(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			data := []byte("%PDF-1.4\n%%EOF")
			upload := tests.AssertRequest(cl.AssetUploadWithBodyWithResponse(
				root,
				&openapi.AssetUploadParams{ContentLength: int64(len(data)), Filename: opt.New("klausur.pdf").Ptr()},
				"application/octet-stream",
				bytes.NewReader(data),
				session,
			))(t, http.StatusOK)
			assetID := upload.JSON200.Id

			garbage := []string{assetID, upload.JSON200.Filename, "dqw4w9wgxcq", "this-is-not-an-xid-at-all", "", "d9n2uhh3fmss73bsumm0-missing-pdf"}

			created := tests.AssertRequest(cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
				Title:      "asset ids",
				Body:       opt.New("<p>body</p>").Ptr(),
				Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
				AssetIds:   &garbage,
			}, session))(t, http.StatusOK)
			r.Len(created.JSON200.Assets, 1)
			r.Equal(assetID, created.JSON200.Assets[0].Id)

			updated := tests.AssertRequest(cl.ThreadUpdateWithResponse(root, created.JSON200.Slug, openapi.ThreadMutableProps{
				Body:     opt.New(`<p><a href="https://youtu.be/dQw4w9WgXcQ">video</a></p>`).Ptr(),
				AssetIds: &garbage,
			}, session))(t, http.StatusOK)
			r.Len(updated.JSON200.Assets, 1)
			r.Equal(assetID, updated.JSON200.Assets[0].Id)

			reply := tests.AssertRequest(cl.ReplyCreateWithResponse(root, created.JSON200.Slug, openapi.ReplyInitialProps{
				Body:     "<p>reply</p>",
				AssetIds: &garbage,
			}, session))(t, http.StatusOK)
			r.Len(reply.JSON200.Assets, 1)
			r.Equal(assetID, reply.JSON200.Assets[0].Id)
		}))
	}))
}
