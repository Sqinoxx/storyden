package drive_test

import (
	"context"
	"net/http"
	"testing"

	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

// The service account can read every folder shared with it, which is a much
// larger set than the folders an administrator registered here. These tests
// pin down the boundary: knowing a Drive ID must not be enough to reach it.
func TestDriveAncestryBoundary(t *testing.T) {
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

			// Only the course folder is registered. "Private" and everything
			// under it stays outside the site.
			folderID := createFolder(t, cl, adminCtx, admin, "Course material", rootDriveID, "public")

			t.Run("cannot_list_a_folder_outside_the_root", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, folderID, &openapi.DriveFolderContentsParams{
						ChildId: ptr(outsideRootID),
					}),
				)(t, http.StatusNotFound)
			})

			t.Run("cannot_download_a_file_outside_the_root", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, outsideFileID, nil),
				)(t, http.StatusNotFound)
			})

			t.Run("cannot_download_via_an_unknown_id", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, "nosuchfile000000001", nil),
				)(t, http.StatusNotFound)
			})

			t.Run("even_an_admin_cannot_escape_the_root", func(t *testing.T) {
				// Registering a folder grants access to that subtree only. It
				// is not a general grant over the service account's reach.
				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(adminCtx, folderID, outsideFileID, nil, admin),
				)(t, http.StatusNotFound)
			})

			t.Run("descendants_of_the_root_remain_reachable", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, deepFileID, nil),
				)(t, http.StatusOK)
			})

			t.Run("a_file_is_not_listable_as_a_folder", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, folderID, &openapi.DriveFolderContentsParams{
						ChildId: ptr(notesFileID),
					}),
				)(t, http.StatusBadRequest)
			})

			t.Run("a_folder_is_not_downloadable", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, subDriveID, nil),
				)(t, http.StatusBadRequest)
			})

			t.Run("deleted_folder_stops_serving_content", func(t *testing.T) {
				tests.AssertRequest(
					cl.AdminDriveFolderDeleteWithResponse(adminCtx, folderID, admin),
				)(t, http.StatusOK)

				tests.AssertRequest(
					cl.DriveFileDownloadWithResponse(root, folderID, notesFileID, nil),
				)(t, http.StatusNotFound)

				tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, folderID, nil),
				)(t, http.StatusNotFound)
			})
		}))
	}))
}
