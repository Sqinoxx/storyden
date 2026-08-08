package drive_test

import (
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

func TestDriveAdminCRUD(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), withFakeDrive(fakeDrive()), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			t.Run("create_list_update_delete", func(t *testing.T) {
				r := require.New(t)

				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				admin := sh.WithSession(adminCtx)

				created := tests.AssertRequest(
					cl.AdminDriveFolderCreateWithResponse(adminCtx, openapi.AdminDriveFolderCreateJSONRequestBody{
						Name:        "Course material",
						Link:        "https://drive.google.com/drive/folders/" + rootDriveID,
						Description: opt.New("Slides and past papers").Ptr(),
						Visibility:  ptr(openapi.DriveVisibility("member")),
					}, admin),
				)(t, http.StatusOK)

				r.NotNil(created.JSON200)
				r.Equal("Course material", created.JSON200.Name)
				r.Equal(rootDriveID, created.JSON200.DriveFolderId)
				r.Equal(openapi.DriveVisibility("member"), created.JSON200.Visibility)

				id := created.JSON200.Id

				listed := tests.AssertRequest(
					cl.AdminDriveFolderListWithResponse(adminCtx, admin),
				)(t, http.StatusOK)

				r.NotNil(listed.JSON200)
				r.Len(listed.JSON200.Folders, 1)
				r.Equal(id, listed.JSON200.Folders[0].Id)

				updated := tests.AssertRequest(
					cl.AdminDriveFolderUpdateWithResponse(adminCtx, id, openapi.AdminDriveFolderUpdateJSONRequestBody{
						Name:       opt.New("Renamed").Ptr(),
						Visibility: ptr(openapi.DriveVisibility("admin")),
					}, admin),
				)(t, http.StatusOK)

				r.NotNil(updated.JSON200)
				r.Equal("Renamed", updated.JSON200.Name)
				r.Equal(openapi.DriveVisibility("admin"), updated.JSON200.Visibility)
				// The Drive folder itself is untouched by a metadata-only edit.
				r.Equal(rootDriveID, updated.JSON200.DriveFolderId)

				tests.AssertRequest(
					cl.AdminDriveFolderDeleteWithResponse(adminCtx, id, admin),
				)(t, http.StatusOK)

				afterDelete := tests.AssertRequest(
					cl.AdminDriveFolderListWithResponse(adminCtx, admin),
				)(t, http.StatusOK)

				r.NotNil(afterDelete.JSON200)
				r.Empty(afterDelete.JSON200.Folders)
			})

			t.Run("rejects_invalid_link", func(t *testing.T) {
				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_002_Frigg)
				admin := sh.WithSession(adminCtx)

				tests.AssertRequest(
					cl.AdminDriveFolderCreateWithResponse(adminCtx, openapi.AdminDriveFolderCreateJSONRequestBody{
						Name: "Nonsense",
						Link: "https://drive.google.com/drive/my-drive",
					}, admin),
				)(t, http.StatusBadRequest)
			})

			t.Run("rejects_folder_not_shared_with_service_account", func(t *testing.T) {
				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_002_Frigg)
				admin := sh.WithSession(adminCtx)

				tests.AssertRequest(
					cl.AdminDriveFolderCreateWithResponse(adminCtx, openapi.AdminDriveFolderCreateJSONRequestBody{
						Name: "Unshared",
						Link: "https://drive.google.com/drive/folders/neverSharedWithUs01",
					}, admin),
				)(t, http.StatusBadRequest)
			})

			t.Run("rejects_a_file_link", func(t *testing.T) {
				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_002_Frigg)
				admin := sh.WithSession(adminCtx)

				tests.AssertRequest(
					cl.AdminDriveFolderCreateWithResponse(adminCtx, openapi.AdminDriveFolderCreateJSONRequestBody{
						Name: "A file, not a folder",
						Link: notesFileID,
					}, admin),
				)(t, http.StatusBadRequest)
			})

			t.Run("denies_non_admin", func(t *testing.T) {
				memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
				member := sh.WithSession(memberCtx)

				tests.AssertRequest(
					cl.AdminDriveFolderListWithResponse(memberCtx, member),
				)(t, http.StatusForbidden)

				tests.AssertRequest(
					cl.AdminDriveFolderCreateWithResponse(memberCtx, openapi.AdminDriveFolderCreateJSONRequestBody{
						Name: "Sneaky",
						Link: rootDriveID,
					}, member),
				)(t, http.StatusForbidden)
			})

			t.Run("denies_anonymous", func(t *testing.T) {
				tests.AssertRequest(
					cl.AdminDriveFolderListWithResponse(root),
				)(t, http.StatusUnauthorized)
			})
		}))
	}))
}

func ptr[T any](v T) *T { return &v }
