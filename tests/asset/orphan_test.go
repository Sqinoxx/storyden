package asset_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/Southclaws/fault"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/ent"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

var errStorageFull = fault.New("storage is full")

// failingStorer stands in for a full disk, an S3 outage or a client that hung
// up mid-body: the write fails after the request was already accepted.
type failingStorer struct {
	object.Storer
}

func (failingStorer) Write(ctx context.Context, path string, r io.Reader, size int64) error {
	return errStorageFull
}

// TestAsset_FailedWriteLeavesNoOrphanRow covers the failure mode that poisoned
// the OCR backfill: an asset row whose bytes were never stored. Such a row can
// never be served (AssetGet 404s out of storage) and cannot be repaired, so it
// must not be created at all.
func TestAsset_FailedWriteLeavesNoOrphanRow(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(),
		fx.Decorate(func(s object.Storer) object.Storer { return failingStorer{s} }),
		fx.Invoke(func(
			root context.Context,
			lc fx.Lifecycle,
			cl *openapi.ClientWithResponses,
			sh *e2e.SessionHelper,
			aw *account_writer.Writer,
			db *ent.Client,
		) {
			lc.Append(fx.StartHook(func() {
				r := require.New(t)

				ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				session := sh.WithSession(ctx)

				before, err := db.Asset.Query().Count(root)
				r.NoError(err)

				filename := "doomed.png"
				resp, err := cl.AssetUploadWithBodyWithResponse(
					root,
					&openapi.AssetUploadParams{
						ContentLength: int64(len(onePixelPNG)),
						Filename:      &filename,
					},
					"image/png",
					bytes.NewReader(onePixelPNG),
					session,
				)
				r.NoError(err)
				r.NotEqual(http.StatusOK, resp.StatusCode(), "a failed object write must not report success")

				after, err := db.Asset.Query().Count(root)
				r.NoError(err)
				r.Equal(before, after, "no asset row may survive a failed object write")
			}))
		}),
	)
}
