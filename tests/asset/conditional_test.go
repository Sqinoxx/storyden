package asset_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
)

func withHeader(name, value string) openapi.RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		req.Header.Set(name, value)
		return nil
	}
}

// TestAsset_ConditionalRequest covers revalidation. Asset bytes are immutable so
// a client holding the current validator must get a bodyless 304 rather than a
// second full transfer.
func TestAsset_ConditionalRequest(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			a := uploadNamedAsset(t, root, cl, session, "image/png", "cached.png", onePixelPNG)

			first, err := cl.AssetGetWithResponse(root, a.Filename)
			r.NoError(err)
			r.Equal(http.StatusOK, first.StatusCode())

			etag := first.HTTPResponse.Header.Get("ETag")
			r.NotEmpty(etag)

			second, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("If-None-Match", etag))
			r.NoError(err)
			r.Equal(http.StatusNotModified, second.StatusCode())
			r.Empty(second.Body, "a 304 must not carry a body")
			r.Equal(etag, second.HTTPResponse.Header.Get("ETag"))

			lastModified := first.HTTPResponse.Header.Get("Last-Modified")
			third, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("If-Modified-Since", lastModified))
			r.NoError(err)
			r.Equal(http.StatusNotModified, third.StatusCode())
		}))
	}))
}

// TestAsset_StaleValidatorStillSendsBody guards the other direction: a client
// holding an older validator must not be fobbed off with a 304.
func TestAsset_StaleValidatorStillSendsBody(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			a := uploadNamedAsset(t, root, cl, session, "image/png", "fresh.png", onePixelPNG)

			get, err := cl.AssetGetWithResponse(root, a.Filename,
				withHeader("If-Modified-Since", "Mon, 02 Jan 2006 15:04:05 GMT"))
			r.NoError(err)
			r.Equal(http.StatusOK, get.StatusCode())
			r.Equal(onePixelPNG, get.Body)
		}))
	}))
}

// TestAsset_RangeRequest covers seeking, which PDF viewers rely on. Without it
// every jump inside a document re-downloads the whole file.
func TestAsset_RangeRequest(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			a := uploadNamedAsset(t, root, cl, session, "image/png", "seekable.png", onePixelPNG)
			total := len(onePixelPNG)

			t.Run("closed range", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("Range", "bytes=0-9"))
				r.NoError(err)
				r.Equal(http.StatusPartialContent, get.StatusCode())
				r.Equal(onePixelPNG[0:10], get.Body)
				r.Equal(fmt.Sprintf("bytes 0-9/%d", total), get.HTTPResponse.Header.Get("Content-Range"))
			})

			t.Run("open ended range", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("Range", "bytes=8-"))
				r.NoError(err)
				r.Equal(http.StatusPartialContent, get.StatusCode())
				r.Equal(onePixelPNG[8:], get.Body)
				r.Equal(fmt.Sprintf("bytes 8-%d/%d", total-1, total), get.HTTPResponse.Header.Get("Content-Range"))
			})

			t.Run("suffix range", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("Range", "bytes=-4"))
				r.NoError(err)
				r.Equal(http.StatusPartialContent, get.StatusCode())
				r.Equal(onePixelPNG[total-4:], get.Body)
			})

			t.Run("beyond the end", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename,
					withHeader("Range", fmt.Sprintf("bytes=%d-", total+100)))
				r.NoError(err)
				r.Equal(http.StatusRequestedRangeNotSatisfiable, get.StatusCode())
				r.Equal(fmt.Sprintf("bytes */%d", total), get.HTTPResponse.Header.Get("Content-Range"))
			})

			// Multipart ranges are not implemented. RFC 7233 allows ignoring
			// the header instead of refusing, so the client still gets usable
			// data rather than a 416 it may not recover from.
			t.Run("unsupported multi range falls back to the full body", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("Range", "bytes=0-1,4-5"))
				r.NoError(err)
				r.Equal(http.StatusOK, get.StatusCode())
				r.Equal(onePixelPNG, get.Body)
			})

			t.Run("malformed range is ignored", func(t *testing.T) {
				r := require.New(t)

				get, err := cl.AssetGetWithResponse(root, a.Filename, withHeader("Range", "kilometres=1-2"))
				r.NoError(err)
				r.Equal(http.StatusOK, get.StatusCode())
				r.Equal(onePixelPNG, get.Body)
			})
		}))
	}))
}
