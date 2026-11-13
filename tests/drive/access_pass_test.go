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

func TestDriveAccessPass(t *testing.T) {
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
			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			admin := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)

			folderID := createFolder(t, cl, adminCtx, admin, "Altprotokolle", rootDriveID, "member")

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:           "Z1-Protokolle",
				PriceLabel:     opt.New("10 €").Ptr(),
				DriveFolderIds: &[]openapi.Identifier{folderID},
			}, admin))(t, http.StatusOK).JSON200
			r.Equal([]openapi.Identifier{folderID}, pass.DriveFolderIds)

			t.Run("listed_with_lock", func(t *testing.T) {
				r := require.New(t)
				list := tests.AssertRequest(cl.DriveFolderListWithResponse(root, memberSession))(t, http.StatusOK).JSON200
				r.Len(list.Folders, 1)
				r.NotNil(list.Folders[0].Access)
				r.True(list.Folders[0].Access.Locked)
				r.Equal("Z1-Protokolle", list.Folders[0].Access.Passes[0].Name)

				adminList := tests.AssertRequest(cl.DriveFolderListWithResponse(root, admin))(t, http.StatusOK).JSON200
				r.Nil(adminList.Folders[0].Access, "admins bypass the lock")
			})

			t.Run("contents_and_files_withheld", func(t *testing.T) {
				r := require.New(t)
				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, folderID, nil, memberSession))(t, http.StatusForbidden)

				resp, err := cl.DriveFileDownloadWithResponse(root, folderID, notesFileID, nil, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, resp.StatusCode())
				r.NotContains(string(resp.Body), notesContent)

				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, folderID, nil, admin))(t, http.StatusOK)
			})

			t.Run("unlocked_after_grant", func(t *testing.T) {
				r := require.New(t)
				tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{}, admin))(t, http.StatusOK)

				list := tests.AssertRequest(cl.DriveFolderListWithResponse(root, memberSession))(t, http.StatusOK).JSON200
				r.Nil(list.Folders[0].Access)

				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, folderID, nil, memberSession))(t, http.StatusOK)
				resp := tests.AssertRequest(cl.DriveFileDownloadWithResponse(root, folderID, notesFileID, nil, memberSession))(t, http.StatusOK)
				r.Equal(notesContent, string(resp.Body))
			})

			t.Run("detaching_folder_unlocks", func(t *testing.T) {
				r := require.New(t)
				tests.AssertRequest(cl.AccessGrantRemoveWithResponse(root, pass.Id, member.Handle, &openapi.AccessGrantRemoveParams{Action: openapi.Revoke}, admin))(t, http.StatusOK)
				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, folderID, nil, memberSession))(t, http.StatusForbidden)

				updated := tests.AssertRequest(cl.AccessPassUpdateWithResponse(root, pass.Id, openapi.AccessPassMutableProps{
					DriveFolderIds: &[]openapi.Identifier{},
				}, admin))(t, http.StatusOK).JSON200
				r.Empty(updated.DriveFolderIds)

				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, folderID, nil, memberSession))(t, http.StatusOK)
			})
		}))
	}))
}

func TestDriveAccessPassNestedFolder(t *testing.T) {
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
			memberCtx, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			admin := sh.WithSession(adminCtx)
			memberSession := sh.WithSession(memberCtx)

			outer := createFolder(t, cl, adminCtx, admin, "Kursmaterial", rootDriveID, "member")
			inner := createFolder(t, cl, adminCtx, admin, "Altprotokolle", subDriveID, "member")

			pass := tests.AssertRequest(cl.AccessPassCreateWithResponse(root, openapi.AccessPassInitialProps{
				Name:           "Z1-Protokolle",
				DriveFolderIds: &[]openapi.Identifier{inner},
			}, admin))(t, http.StatusOK).JSON200

			throughOuter := &openapi.DriveFolderContentsParams{ChildId: ptr(subDriveID)}

			t.Run("locked_folder_not_reachable_through_open_parent", func(t *testing.T) {
				r := require.New(t)

				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, outer, nil, memberSession))(t, http.StatusOK)
				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, outer, throughOuter, memberSession))(t, http.StatusForbidden)

				resp, err := cl.DriveFileDownloadWithResponse(root, outer, deepFileID, nil, memberSession)
				r.NoError(err)
				r.Equal(http.StatusForbidden, resp.StatusCode())
				r.NotContains(string(resp.Body), deepContent)

				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, outer, throughOuter, admin))(t, http.StatusOK)
			})

			t.Run("google_id_hidden_while_locked", func(t *testing.T) {
				r := require.New(t)
				list := tests.AssertRequest(cl.DriveFolderListWithResponse(root, memberSession))(t, http.StatusOK).JSON200
				for _, f := range list.Folders {
					if f.Id == inner {
						r.Empty(f.DriveFolderId)
					} else {
						r.NotEmpty(f.DriveFolderId)
					}
				}
			})

			t.Run("reachable_after_grant", func(t *testing.T) {
				tests.AssertRequest(cl.AccessGrantSetWithResponse(root, pass.Id, member.Handle, openapi.AccessGrantMutableProps{}, admin))(t, http.StatusOK)
				tests.AssertRequest(cl.DriveFolderContentsWithResponse(root, outer, throughOuter, memberSession))(t, http.StatusOK)
				resp := tests.AssertRequest(cl.DriveFileDownloadWithResponse(root, outer, deepFileID, nil, memberSession))(t, http.StatusOK)
				r.Equal(deepContent, string(resp.Body))
			})
		}))
	}))
}
