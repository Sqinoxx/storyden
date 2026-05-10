package account

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/account/account_writer"
	"github.com/Southclaws/storyden/app/resources/seed"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
	"github.com/Southclaws/storyden/internal/integration"
	"github.com/Southclaws/storyden/internal/integration/e2e"
	"github.com/Southclaws/storyden/tests"
)

func TestAccountCreateProvisioning(t *testing.T) {
	if tests.IsSharedPostgresDatabase() {
		t.Skip("skipping account provisioning test on shared postgres database")
	}

	t.Parallel()

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
			memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
			memberSession := sh.WithSession(memberCtx)

			accessKey := tests.AssertRequest(
				cl.AccessKeyCreateWithResponse(root, openapi.AccessKeyInitialProps{
					Name: "account-provisioning-" + xid.New().String(),
				}, adminSession),
			)(t, http.StatusOK)
			accessKeySession := createAccessKeyAuth(accessKey.JSON200.Secret)

			unauthenticated, err := cl.AccountManageCreateWithResponse(root, openapi.AccountManageCreateJSONRequestBody{
				Handle: openapi.AccountHandle(xid.New().String()),
			})
			tests.Status(t, err, unauthenticated, http.StatusUnauthorized)

			forbidden, err := cl.AccountManageCreateWithResponse(root, openapi.AccountManageCreateJSONRequestBody{
				Handle: openapi.AccountHandle(xid.New().String()),
			}, memberSession)
			tests.Status(t, err, forbidden, http.StatusForbidden)

			registrationMode := openapi.RegistrationModeDisabled
			tests.AssertRequest(
				cl.AdminSettingsUpdateWithResponse(root, openapi.AdminSettingsUpdateJSONRequestBody{
					RegistrationMode: &registrationMode,
				}, adminSession),
			)(t, http.StatusOK)

			email := openapi.EmailAddress(fmt.Sprintf("%s@example.com", xid.New().String()))
			name := openapi.AccountName("Provisioned Account")
			handle := openapi.AccountHandle(xid.New().String())

			created := tests.AssertRequest(
				cl.AccountManageCreateWithResponse(root, openapi.AccountManageCreateJSONRequestBody{
					Handle:       handle,
					Name:         &name,
					EmailAddress: &email,
				}, accessKeySession),
			)(t, http.StatusOK)
			require.NotNil(t, created.JSON200)
			assert.Equal(t, name, created.JSON200.Name)
			require.Len(t, created.JSON200.EmailAddresses, 1)
			assert.Equal(t, email, created.JSON200.EmailAddresses[0].EmailAddress)
			assert.False(t, created.JSON200.EmailAddresses[0].Verified)

			duplicateHandle, err := cl.AccountManageCreateWithResponse(root, openapi.AccountManageCreateJSONRequestBody{
				Handle: handle,
			}, adminSession)
			tests.Status(t, err, duplicateHandle, http.StatusConflict)

			duplicateEmailHandle := openapi.AccountHandle(xid.New().String())
			duplicateEmail, err := cl.AccountManageCreateWithResponse(root, openapi.AccountManageCreateJSONRequestBody{
				Handle:       duplicateEmailHandle,
				EmailAddress: &email,
			}, adminSession)
			tests.Status(t, err, duplicateEmail, http.StatusConflict)
		}))
	}))
}

func createAccessKeyAuth(accessKeyToken string) openapi.RequestEditorFn {
	authHeader := fmt.Sprintf("Bearer %s", accessKeyToken)

	return func(ctx context.Context, req *http.Request) error {
		req.Header.Set("Authorization", authHeader)
		return nil
	}
}
