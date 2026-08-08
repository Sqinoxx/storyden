package drive_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/dt"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestDriveBrowseVisibility(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), withFakeDrive(fakeDrive()), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			r := require.New(t)

			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			admin := sh.WithSession(adminCtx)

			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			member := sh.WithSession(memberCtx)

			publicFolder := createFolder(t, cl, adminCtx, admin, "Open shelf", rootDriveID, "public")
			memberFolder := createFolder(t, cl, adminCtx, admin, "Members shelf", subDriveID, "member")
			adminFolder := createFolder(t, cl, adminCtx, admin, "Staff shelf", outsideRootID, "admin")

			t.Run("guest_sees_only_public", func(t *testing.T) {
				resp := tests.AssertRequest(cl.DriveFolderListWithResponse(root))(t, http.StatusOK)

				r.NotNil(resp.JSON200)
				require.Equal(t, []string{publicFolder}, folderIDs(resp.JSON200.Folders))
			})

			t.Run("member_sees_public_and_member", func(t *testing.T) {
				resp := tests.AssertRequest(cl.DriveFolderListWithResponse(memberCtx, member))(t, http.StatusOK)

				r.NotNil(resp.JSON200)
				require.ElementsMatch(t, []string{publicFolder, memberFolder}, folderIDs(resp.JSON200.Folders))
			})

			t.Run("admin_sees_all", func(t *testing.T) {
				resp := tests.AssertRequest(cl.DriveFolderListWithResponse(adminCtx, admin))(t, http.StatusOK)

				r.NotNil(resp.JSON200)
				require.ElementsMatch(t,
					[]string{publicFolder, memberFolder, adminFolder},
					folderIDs(resp.JSON200.Folders))
			})

			t.Run("member_cannot_open_admin_folder", func(t *testing.T) {
				// Reported as missing, not forbidden, so that the folder's
				// existence stays invisible to members.
				tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(memberCtx, adminFolder, nil, member),
				)(t, http.StatusNotFound)
			})

			t.Run("guest_cannot_open_member_folder", func(t *testing.T) {
				tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, memberFolder, nil),
				)(t, http.StatusNotFound)
			})
		}))
	}))
}

func TestDriveBrowseNavigation(t *testing.T) {
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

			t.Run("lists_root_contents", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, folderID, nil),
				)(t, http.StatusOK)

				r.NotNil(resp.JSON200)

				// Folders sort ahead of files, matching Drive's own ordering.
				r.Equal([]string{subDriveID, nativeDocID, notesFileID}, entryIDs(resp.JSON200.Entries))

				// The root crumb carries the administrator's name for the
				// folder, not the name it has inside Drive.
				r.Len(resp.JSON200.Breadcrumbs, 1)
				r.Equal("Course material", resp.JSON200.Breadcrumbs[0].Name)

				sub := findEntry(t, resp.JSON200.Entries, subDriveID)
				r.True(sub.IsFolder)

				notes := findEntry(t, resp.JSON200.Entries, notesFileID)
				r.False(notes.IsFolder)
				r.Equal("text/plain", notes.MimeType)
				r.True(notes.Previewable)
				r.NotNil(notes.Size)
				r.Equal(int64(len(notesContent)), *notes.Size)

				// A Google Doc is reported as the PDF it will be exported to,
				// so a client can decide to preview it without knowing about
				// Google's export rules.
				native := findEntry(t, resp.JSON200.Entries, nativeDocID)
				r.Equal("application/pdf", native.MimeType)
				r.True(native.Previewable)
			})

			t.Run("descends_into_subfolder_with_breadcrumbs", func(t *testing.T) {
				r := require.New(t)

				resp := tests.AssertRequest(
					cl.DriveFolderContentsWithResponse(root, folderID, &openapi.DriveFolderContentsParams{
						ChildId: ptr(subDriveID),
					}),
				)(t, http.StatusOK)

				r.NotNil(resp.JSON200)
				r.Equal([]string{deepFileID}, entryIDs(resp.JSON200.Entries))

				r.Len(resp.JSON200.Breadcrumbs, 2)
				r.Equal("Course material", resp.JSON200.Breadcrumbs[0].Name)
				r.Equal(subDriveID, resp.JSON200.Breadcrumbs[1].Id)
				r.Equal("Semester 1", resp.JSON200.Breadcrumbs[1].Name)
			})
		}))
	}))
}

func createFolder(
	t *testing.T,
	cl *openapi.ClientWithResponses,
	ctx context.Context,
	session openapi.RequestEditorFn,
	name string,
	driveID string,
	visibility string,
) string {
	t.Helper()

	resp := tests.AssertRequest(
		cl.AdminDriveFolderCreateWithResponse(ctx, openapi.AdminDriveFolderCreateJSONRequestBody{
			Name:       name,
			Link:       driveID,
			Visibility: ptr(openapi.DriveVisibility(visibility)),
		}, session),
	)(t, http.StatusOK)

	require.NotNil(t, resp.JSON200)

	return resp.JSON200.Id
}

func folderIDs(in []openapi.DriveFolder) []string {
	return dt.Map(in, func(f openapi.DriveFolder) string { return f.Id })
}

func entryIDs(in []openapi.DriveEntry) []string {
	return dt.Map(in, func(e openapi.DriveEntry) string { return e.Id })
}

func findEntry(t *testing.T, entries []openapi.DriveEntry, id string) openapi.DriveEntry {
	t.Helper()

	for _, e := range entries {
		if e.Id == id {
			return e
		}
	}

	t.Fatalf("entry %s not found in listing", id)

	return openapi.DriveEntry{}
}
