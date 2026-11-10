package tag_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/Southclaws/opt"
	"github.com/rs/xid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/account/role"
	"github.com/Southclaws/storyden/app/resources/rbac"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/internal/utils"
	"github.com/Southclaws/storyden/tests"
)

func TestTagPermissions(t *testing.T) {
	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
			adminSession := sh.WithSession(adminCtx)

			memberPerms := lo.FilterMap(role.DefaultRoleMember.Permissions.List(), func(p rbac.Permission, _ int) (openapi.Permission, bool) {
				return openapi.Permission(p.String()), p != rbac.PermissionCreateTags
			})
			tests.AssertRequest(cl.RoleUpdateWithResponse(root, role.DefaultRoleMemberID.String(), openapi.RoleMutableProps{
				Permissions: &memberPerms,
			}, adminSession))(t, http.StatusOK)

			cat := tests.AssertRequest(cl.CategoryCreateWithResponse(root, openapi.CategoryInitialProps{
				Name: "tags " + xid.New().String(),
			}, adminSession))(t, http.StatusOK)

			handle := xid.New().String()
			signup := tests.AssertRequest(cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: handle, Token: "password"}))(t, http.StatusOK)
			memberSession := sh.WithSession(e2e.WithAccountID(root, account.AccountID(utils.Must(xid.FromString(signup.JSON200.Id)))))

			existing := xid.New().String()
			tests.AssertRequest(cl.TagCreateWithResponse(root, openapi.TagCreateProps{Name: existing}, adminSession))(t, http.StatusOK)

			createThread := func(session openapi.RequestEditorFn, tags ...string) (*openapi.ThreadCreateResponse, error) {
				return cl.ThreadCreateWithResponse(root, openapi.ThreadInitialProps{
					Title:      "thread " + xid.New().String(),
					Body:       opt.New("<p>tagged</p>").Ptr(),
					Category:   opt.New(cat.JSON200.Id).Ptr(),
					Visibility: opt.New(openapi.VisibilityPublished).Ptr(),
					Tags:       &tags,
				}, session)
			}

			createNode := func(session openapi.RequestEditorFn, tags ...string) (*openapi.NodeCreateResponse, error) {
				return cl.NodeCreateWithResponse(root, openapi.NodeInitialProps{
					Name: "node " + xid.New().String(),
					Tags: &tags,
				}, session)
			}

			t.Run("without_permission", func(t *testing.T) {
				tests.AssertRequest(cl.TagCreateWithResponse(root, openapi.TagCreateProps{Name: xid.New().String()}, memberSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.TagUpdateWithResponse(root, existing, openapi.TagCreateProps{Name: xid.New().String()}, memberSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.TagDeleteWithResponse(root, existing, memberSession))(t, http.StatusForbidden)

				newTag := xid.New().String()
				tests.AssertRequest(createThread(memberSession, newTag))(t, http.StatusForbidden)
				tests.AssertRequest(createThread(memberSession, existing, newTag))(t, http.StatusForbidden)
				tests.AssertRequest(createNode(memberSession, newTag))(t, http.StatusForbidden)

				tests.AssertRequest(cl.TagGetWithResponse(root, newTag, adminSession))(t, http.StatusNotFound)

				thr := tests.AssertRequest(createThread(memberSession, existing))(t, http.StatusOK)
				assert.Equal(t, []string{existing}, dtNames(thr.JSON200.Tags))

				node := tests.AssertRequest(createNode(memberSession, existing))(t, http.StatusOK)
				assert.Equal(t, []string{existing}, dtNames(node.JSON200.Tags))

				tests.AssertRequest(cl.ThreadUpdateWithResponse(root, thr.JSON200.Slug, openapi.ThreadMutableProps{
					Tags: &[]string{existing, xid.New().String()},
				}, memberSession))(t, http.StatusForbidden)
			})

			taggerRole := tests.AssertRequest(cl.RoleCreateWithResponse(root, openapi.RoleInitialProps{
				Name:        "tagger " + xid.New().String(),
				Colour:      "blue",
				Permissions: []openapi.Permission{openapi.CREATETAGS},
			}, adminSession))(t, http.StatusOK)
			tests.AssertRequest(cl.AccountAddRoleWithResponse(root, handle, taggerRole.JSON200.Id, adminSession))(t, http.StatusOK)

			t.Run("with_create_tags", func(t *testing.T) {
				newTag := xid.New().String()
				thr := tests.AssertRequest(createThread(memberSession, existing, newTag))(t, http.StatusOK)
				assert.ElementsMatch(t, []string{existing, newTag}, dtNames(thr.JSON200.Tags))

				nodeTag := xid.New().String()
				node := tests.AssertRequest(createNode(memberSession, nodeTag))(t, http.StatusOK)
				assert.Equal(t, []string{nodeTag}, dtNames(node.JSON200.Tags))

				tests.AssertRequest(cl.TagCreateWithResponse(root, openapi.TagCreateProps{Name: xid.New().String()}, memberSession))(t, http.StatusOK)

				tests.AssertRequest(cl.TagUpdateWithResponse(root, existing, openapi.TagCreateProps{Name: xid.New().String()}, memberSession))(t, http.StatusForbidden)
				tests.AssertRequest(cl.TagDeleteWithResponse(root, existing, memberSession))(t, http.StatusForbidden)
			})

			managerRole := tests.AssertRequest(cl.RoleCreateWithResponse(root, openapi.RoleInitialProps{
				Name:        "tag manager " + xid.New().String(),
				Colour:      "blue",
				Permissions: []openapi.Permission{openapi.MANAGETAGS},
			}, adminSession))(t, http.StatusOK)

			managerHandle := xid.New().String()
			managerSignup := tests.AssertRequest(cl.AuthPasswordSignupWithResponse(root, nil, openapi.AuthPair{Identifier: managerHandle, Token: "password"}))(t, http.StatusOK)
			managerSession := sh.WithSession(e2e.WithAccountID(root, account.AccountID(utils.Must(xid.FromString(managerSignup.JSON200.Id)))))
			tests.AssertRequest(cl.AccountAddRoleWithResponse(root, managerHandle, managerRole.JSON200.Id, adminSession))(t, http.StatusOK)

			t.Run("with_manage_tags", func(t *testing.T) {
				created := tests.AssertRequest(cl.TagCreateWithResponse(root, openapi.TagCreateProps{Name: xid.New().String()}, managerSession))(t, http.StatusOK)

				newTag := xid.New().String()
				tests.AssertRequest(createThread(managerSession, newTag))(t, http.StatusOK)

				renamed := xid.New().String()
				tests.AssertRequest(cl.TagUpdateWithResponse(root, created.JSON200.Name, openapi.TagCreateProps{Name: renamed}, managerSession))(t, http.StatusOK)
				tests.AssertRequest(cl.TagDeleteWithResponse(root, renamed, managerSession))(t, http.StatusOK)
				tests.AssertRequest(cl.TagGetWithResponse(root, renamed, managerSession))(t, http.StatusNotFound)
			})
		}))
	}))
}

func dtNames(tags []openapi.TagReference) []string {
	return lo.Map(tags, func(t openapi.TagReference, _ int) string { return t.Name })
}
