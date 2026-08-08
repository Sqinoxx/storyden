package asset_test

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// onePixelPNG is a real, decodable PNG. Uploads sniff content and the branding
// pipeline actually decodes it, so a hand-written byte blob with a stale
// checksum is not good enough.
var onePixelPNG = encodePNG(1, 1)

func encodePNG(w, h int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{R: uint8(x * 8), G: uint8(y * 8), B: 128, A: 255})
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		panic(err)
	}

	return buf.Bytes()
}

func noOCR() *config.Config {
	return &config.Config{OCREnabled: false, OCRBackfillEnabled: false}
}

// TestAsset_UploadDownloadRoundTrip pins the contract the whole frontend
// depends on: what goes in comes back out byte for byte, described correctly.
func TestAsset_UploadDownloadRoundTrip(t *testing.T) {
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

			a := uploadNamedAsset(t, root, cl, session, "image/png", "holiday.png", onePixelPNG)

			r.Equal("image/png", a.MimeType)
			r.Contains(a.Filename, "holiday", "the requested filename should survive into the asset name")
			r.Equal("/api/assets/"+a.Filename, a.Path)

			get, err := cl.AssetGetWithResponse(root, a.Filename)
			r.NoError(err)
			r.Equal(http.StatusOK, get.StatusCode())

			r.Equal(onePixelPNG, get.Body, "downloaded bytes must match what was uploaded")

			h := get.HTTPResponse.Header
			r.Equal("image/png", h.Get("Content-Type"))
			r.Equal(int64(len(onePixelPNG)), get.HTTPResponse.ContentLength)
			r.Equal("nosniff", h.Get("X-Content-Type-Options"))
			r.Equal("bytes", h.Get("Accept-Ranges"))
			r.Contains(h.Get("Content-Disposition"), "inline", "images are safe to render in place")
			r.NotEmpty(h.Get("Cache-Control"))

			r.NotEmpty(h.Get("ETag"), "an empty validator is worse than none")
			r.NotEmpty(h.Get("Last-Modified"))

			_, err = http.ParseTime(h.Get("Last-Modified"))
			r.NoError(err, "Last-Modified must be a spec-compliant http-date")
		}))
	}))
}

// TestAsset_DownloadUnknownFilename guards the 404 path, which is what a stale
// link or a cleaned-up object produces.
func TestAsset_DownloadUnknownFilename(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			get, err := cl.AssetGetWithResponse(root, xid.New().String()+"-nothing-here")
			r.NoError(err)
			r.Equal(http.StatusNotFound, get.StatusCode())
		}))
	}))
}

// TestAsset_UploadWithoutFilename covers the documented fallback: clients may
// omit the filename query and the server invents a placeholder.
func TestAsset_UploadWithoutFilename(t *testing.T) {
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

			a := uploadTestAsset(t, root, cl, session, "image/png", onePixelPNG)
			r.Contains(a.Filename, "untitled")

			get, err := cl.AssetGetWithResponse(root, a.Filename)
			r.NoError(err)
			r.Equal(http.StatusOK, get.StatusCode())
		}))
	}))
}

// TestAsset_OverstatedContentLength covers a client that declares more bytes
// than it sends — a lie, a truncated proxy, or a dropped connection. The stored
// size must describe what is actually on disk, otherwise every later consumer
// (Content-Length, the OCR size gate, range requests) works from a wrong number.
func TestAsset_OverstatedContentLength(t *testing.T) {
	integration.Test(t, noOCR(), e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
		aq *asset_querier.Querier,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			ctx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			session := sh.WithSession(ctx)

			resp, err := cl.AssetUploadWithBodyWithResponse(
				root,
				&openapi.AssetUploadParams{
					ContentLength: int64(len(onePixelPNG)) * 4,
					Filename:      ptr("liar.png"),
				},
				"image/png",
				bytes.NewReader(onePixelPNG),
				session,
			)
			tests.Ok(t, err, resp)

			assetID, err := xid.FromString(resp.JSON200.Id)
			r.NoError(err)

			stored, err := aq.GetByID(root, assetID)
			r.NoError(err)
			r.Equal(len(onePixelPNG), stored.Size, "recorded size must be the bytes actually stored")

			get, err := cl.AssetGetWithResponse(root, resp.JSON200.Filename)
			r.NoError(err)
			r.Equal(http.StatusOK, get.StatusCode())
			r.Equal(onePixelPNG, get.Body)
		}))
	}))
}

// TestAsset_NonRenderableIsForcedToDownload covers the stored-XSS surface: asset
// downloads are public and unauthenticated, so anything a browser might execute
// must arrive as an attachment rather than an active document.
func TestAsset_NonRenderableIsForcedToDownload(t *testing.T) {
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

			page := []byte("<html><body><script>alert(document.domain)</script></body></html>")
			a := uploadNamedAsset(t, root, cl, session, "text/html", "payload.html", page)

			get, err := cl.AssetGetWithResponse(root, a.Filename)
			r.NoError(err)
			r.Equal(http.StatusOK, get.StatusCode())

			h := get.HTTPResponse.Header
			r.Contains(h.Get("Content-Disposition"), "attachment")
			r.Equal("nosniff", h.Get("X-Content-Type-Options"))
		}))
	}))
}

func uploadNamedAsset(t *testing.T, ctx context.Context, cl *openapi.ClientWithResponses, session openapi.RequestEditorFn, contentType string, filename string, data []byte) openapi.Asset {
	t.Helper()

	resp, err := cl.AssetUploadWithBodyWithResponse(
		ctx,
		&openapi.AssetUploadParams{
			ContentLength: int64(len(data)),
			Filename:      &filename,
		},
		contentType,
		bytes.NewReader(data),
		session,
	)
	tests.Ok(t, err, resp)

	return *resp.JSON200
}

func ptr[T any](v T) *T { return &v }
