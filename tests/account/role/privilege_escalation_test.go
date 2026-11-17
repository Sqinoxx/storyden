package role_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/account/role"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestModeratorCannotEscalateToAdministrator(t *testing.T) {
	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, admin := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			modCtx, mod := e2e.WithAccount(root, aw, seed.Account_004_Loki)
			modSession := sh.WithSession(modCtx)

			_, member := e2e.WithAccount(root, aw, seed.Account_003_Baldur)

			modRole := tests.AssertRequest(cl.RoleCreateWithResponse(adminCtx, openapi.RoleCreateJSONRequestBody{
				Name:   "mod-" + xid.New().String(),
				Colour: "red",
				Permissions: openapi.PermissionList{
					openapi.MANAGEROLES,
					openapi.MANAGEACCOUNTS,
					openapi.MANAGECATEGORIES,
				},
			}, adminSession))(t, http.StatusOK)
			tests.AssertRequest(cl.AccountAddRoleWithResponse(adminCtx, mod.Handle, modRole.JSON200.Id, adminSession))(t, http.StatusOK)

			adminRoleID := role.DefaultRoleAdminID.String()

			t.Run("create_role_with_held_permissions", func(t *testing.T) {
				created := tests.AssertRequest(cl.RoleCreateWithResponse(modCtx, openapi.RoleCreateJSONRequestBody{
					Name:        "helper-" + xid.New().String(),
					Colour:      "blue",
					Permissions: openapi.PermissionList{openapi.MANAGECATEGORIES},
				}, modSession))(t, http.StatusOK)

				tests.AssertRequest(cl.AccountAddRoleWithResponse(modCtx, member.Handle, created.JSON200.Id, modSession))(t, http.StatusOK)
			})

			t.Run("create_role_with_administrator", func(t *testing.T) {
				tests.AssertRequest(cl.RoleCreateWithResponse(modCtx, openapi.RoleCreateJSONRequestBody{
					Name:        "sneaky-" + xid.New().String(),
					Colour:      "blue",
					Permissions: openapi.PermissionList{openapi.ADMINISTRATOR},
				}, modSession))(t, http.StatusForbidden)
			})

			t.Run("create_role_with_permission_not_held", func(t *testing.T) {
				tests.AssertRequest(cl.RoleCreateWithResponse(modCtx, openapi.RoleCreateJSONRequestBody{
					Name:        "sneaky-" + xid.New().String(),
					Colour:      "blue",
					Permissions: openapi.PermissionList{openapi.MANAGESETTINGS},
				}, modSession))(t, http.StatusForbidden)
			})

			t.Run("add_administrator_to_own_role", func(t *testing.T) {
				perms := openapi.PermissionList{openapi.MANAGEROLES, openapi.ADMINISTRATOR}
				tests.AssertRequest(cl.RoleUpdateWithResponse(modCtx, modRole.JSON200.Id, openapi.RoleUpdateJSONRequestBody{
					Permissions: &perms,
				}, modSession))(t, http.StatusForbidden)
			})

			t.Run("edit_or_delete_admin_role", func(t *testing.T) {
				name := "renamed"
				tests.AssertRequest(cl.RoleUpdateWithResponse(modCtx, adminRoleID, openapi.RoleUpdateJSONRequestBody{
					Name: &name,
				}, modSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.RoleDeleteWithResponse(modCtx, adminRoleID, modSession))(t, http.StatusForbidden)
			})

			t.Run("assign_admin_role_to_other_account", func(t *testing.T) {
				tests.AssertRequest(cl.AccountAddRoleWithResponse(modCtx, member.Handle, adminRoleID, modSession))(t, http.StatusForbidden)
			})

			t.Run("remove_role_from_administrator", func(t *testing.T) {
				tests.AssertRequest(cl.AccountAddRoleWithResponse(adminCtx, admin.Handle, modRole.JSON200.Id, adminSession))(t, http.StatusOK)
				tests.AssertRequest(cl.AccountRemoveRoleWithResponse(modCtx, admin.Handle, modRole.JSON200.Id, modSession))(t, http.StatusForbidden)
			})

			t.Run("password_reset_token", func(t *testing.T) {
				tests.AssertRequest(cl.AccountPasswordResetTokenGetWithResponse(modCtx, admin.ID.String(), modSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.AccountPasswordResetTokenGetWithResponse(modCtx, member.ID.String(), modSession))(t, http.StatusOK)
				tests.AssertRequest(cl.AccountPasswordResetTokenGetWithResponse(adminCtx, admin.ID.String(), adminSession))(t, http.StatusOK)
			})

			t.Run("manage_administrator_account", func(t *testing.T) {
				bio := "<p>taken over</p>"
				tests.AssertRequest(cl.AccountManageUpdateWithResponse(modCtx, admin.ID.String(), openapi.AccountManageUpdateJSONRequestBody{
					Bio: &bio,
				}, modSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.AccountManageUpdateWithResponse(modCtx, member.ID.String(), openapi.AccountManageUpdateJSONRequestBody{
					Bio: &bio,
				}, modSession))(t, http.StatusOK)
			})

			t.Run("change_administrator_email", func(t *testing.T) {
				tests.AssertRequest(cl.AccountManageUpdateEmailWithResponse(modCtx, admin.ID.String(), xid.New().String(), openapi.AccountManageUpdateEmailJSONRequestBody{
					EmailAddress: "attacker@example.com",
				}, modSession))(t, http.StatusForbidden)
			})
		}))
	}))
}
