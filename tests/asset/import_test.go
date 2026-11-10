package asset_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/asset/asset_import"
	"github.com/Southclaws/storyden/app/services/asset/asset_upload"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

var importPDF = []byte("%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[]/Count 0>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n")

func importSource(t *testing.T) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	mux.HandleFunc("/Leitlinie%20Karies.pdf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write(importPDF)
	})
	mux.HandleFunc("/download", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="Aspirin - Fachinformation.pdf"`)
		w.Write(importPDF)
	})
	mux.HandleFunc("/page", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		io.WriteString(w, "<html><body>Anmelden</body></html>")
	})
	mux.HandleFunc("/notes.txt", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		io.WriteString(w, "nur Text")
	})
	mux.HandleFunc("/huge.pdf", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/pdf")
		w.Write(importPDF)
		w.Write(bytes.Repeat([]byte(" "), 2*1024*1024))
	})
	mux.HandleFunc("/missing.pdf", func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})

	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	return srv
}

func TestAssetImport(t *testing.T) {
	srv := importSource(t)

	integration.Test(t, &config.Config{MaxUploadSizeMB: 1, OCREnabled: false, OCRBackfillEnabled: false}, e2e.Setup(),
		fx.Decorate(func(u *asset_upload.Uploader, s *settings.SettingsRepository, cfg config.Config) *asset_import.Importer {
			return asset_import.NewWithClient(srv.Client(), u, s, cfg)
		}),
		fx.Invoke(func(
			root context.Context,
			lc fx.Lifecycle,
			cl *openapi.ClientWithResponses,
			sh *e2e.SessionHelper,
			aw *account_writer.Writer,
		) {
			lc.Append(fx.StartHook(func() {
				memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
				session := sh.WithSession(memberCtx)

				t.Run("imports a direct pdf link named after its path", func(t *testing.T) {
					r := require.New(t)

					resp, err := cl.AssetImportWithResponse(root, openapi.AssetImportJSONRequestBody{Url: srv.URL + "/Leitlinie%20Karies.pdf"}, session)
					tests.Ok(t, err, resp)
					r.Equal("Leitlinie Karies.pdf", resp.JSON200.Filename)
					r.Equal("application/pdf", resp.JSON200.MimeType)

					get, err := cl.AssetGetWithResponse(root, resp.JSON200.Path[len("/api/assets/"):], nil, session)
					r.NoError(err)
					r.Equal(http.StatusOK, get.StatusCode())
					r.Equal(importPDF, get.Body)
				})

				t.Run("prefers the content disposition filename", func(t *testing.T) {
					resp, err := cl.AssetImportWithResponse(root, openapi.AssetImportJSONRequestBody{Url: srv.URL + "/download"}, session)
					tests.Ok(t, err, resp)
					require.Equal(t, "Aspirin - Fachinformation.pdf", resp.JSON200.Filename)
				})

				for _, c := range []struct {
					name string
					path string
				}{
					{"rejects a web page", "/page"},
					{"rejects other file types", "/notes.txt"},
					{"rejects files over the upload limit", "/huge.pdf"},
					{"rejects missing files", "/missing.pdf"},
				} {
					t.Run(c.name, func(t *testing.T) {
						resp, err := cl.AssetImportWithResponse(root, openapi.AssetImportJSONRequestBody{Url: srv.URL + c.path}, session)
						require.NoError(t, err)
						require.Equal(t, http.StatusBadRequest, resp.StatusCode(), string(resp.Body))
					})
				}

				t.Run("requires a session", func(t *testing.T) {
					resp, err := cl.AssetImportWithResponse(root, openapi.AssetImportJSONRequestBody{Url: srv.URL + "/download"})
					require.NoError(t, err)
					require.Equal(t, http.StatusUnauthorized, resp.StatusCode())
				})
			}))
		}),
	)
}

func TestAssetImport_RefusesPrivateAddresses(t *testing.T) {
	srv := importSource(t)

	integration.Test(t, &config.Config{OCREnabled: false, OCRBackfillEnabled: false}, e2e.Setup(), fx.Invoke(func(
		root context.Context,
		lc fx.Lifecycle,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			session := sh.WithSession(memberCtx)

			for _, u := range []string{srv.URL + "/download", "http://169.254.169.254/latest/meta-data", "file:///etc/passwd"} {
				resp, err := cl.AssetImportWithResponse(root, openapi.AssetImportJSONRequestBody{Url: u}, session)
				require.NoError(t, err)
				require.Equal(t, http.StatusBadRequest, resp.StatusCode(), u)
			}
		}))
	}))
}
