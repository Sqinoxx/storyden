package drive_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestDriveFileDownload(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), withFakeDrive(fakeDrive()), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			admin := sh.WithSession(adminCtx)

			folderID := createFolder(t, cl, adminCtx, admin, "Course material", rootDriveID, "public")

			t.Run("streams_file_bytes", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, notesFileID, nil),
				)(t, http.StatusOK)

				r.Equal(notesContent, string(resp.Body))
				// Drive's own size metadata is not forwarded, so no length is
				// announced at all. A negative or mismatched value here would
				// be an invalid header or a truncated body.
				r.NotEqual("-1", resp.HTTPResponse.Header.Get("Content-Length"))
				r.Equal("nosniff", resp.HTTPResponse.Header.Get("X-Content-Type-Options"))
				r.Equal(`attachment; filename="notes.txt"`, resp.HTTPResponse.Header.Get("Content-Disposition"))
				// Never cacheable by a shared cache: a folder may be restricted
				// to members and the bytes come from an external service.
				r.Contains(resp.HTTPResponse.Header.Get("Cache-Control"), "private")
			})

			t.Run("honours_inline_for_safe_types", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, deepFileID, &openapi.DriveFileDownloadParams{
						Disposition: ptr(openapi.DriveFileDownloadParamsDisposition("inline")),
					}),
				)(t, http.StatusOK)

				r.Equal(deepContent, string(resp.Body))
				r.Equal(`inline; filename="deep.pdf"`, resp.HTTPResponse.Header.Get("Content-Disposition"))
			})

			t.Run("forces_attachment_for_unsafe_types_even_when_inline_asked", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, notesFileID, &openapi.DriveFileDownloadParams{
						Disposition: ptr(openapi.DriveFileDownloadParamsDisposition("inline")),
					}),
				)(t, http.StatusOK)

				r.Equal(`attachment; filename="notes.txt"`, resp.HTTPResponse.Header.Get("Content-Disposition"))
			})

			t.Run("exports_google_native_documents_to_pdf", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, nativeDocID, nil),
				)(t, http.StatusOK)

				r.Contains(resp.HTTPResponse.Header.Get("Content-Type"), "application/pdf")
				r.Contains(string(resp.Body), "%PDF")
			})
		}))
	}))
}
