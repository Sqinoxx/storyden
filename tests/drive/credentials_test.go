package drive_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
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

// fakeServiceAccountKey builds a syntactically valid Google service account
// key with a throwaway RSA key, so the credentials resolver's real parsing
// and client-construction path is exercised without talking to Google.
func fakeServiceAccountKey(t *testing.T, email string) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: mustMarshalPKCS8(t, key),
	}

	b, err := json.Marshal(map[string]string{
		"type":           "service_account",
		"project_id":     "storyden-test",
		"private_key_id": "test-key-id",
		"private_key":    string(pem.EncodeToMemory(block)),
		"client_email":   email,
		"client_id":      "000000000000000000000",
		"token_uri":      "https://oauth2.googleapis.com/token",
	})
	require.NoError(t, err)

	return b
}

func mustMarshalPKCS8(t *testing.T, key *rsa.PrivateKey) []byte {
	t.Helper()

	b, err := x509.MarshalPKCS8PrivateKey(key)
	require.NoError(t, err)

	return b
}

func TestDriveAdminCredentials(t *testing.T) {
	t.Parallel()

	integration.Test(t, nil, e2e.Setup(), fx.Invoke(func(
		lc fx.Lifecycle,
		root context.Context,
		cl *openapi.ClientWithResponses,
		sh *e2e.SessionHelper,
		aw *account_writer.Writer,
	) {
		lc.Append(fx.StartHook(func() {
			t.Run("starts_unconfigured", func(t *testing.T) {
				r := require.New(t)

				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				admin := sh.WithSession(adminCtx)

				status := tests.AssertRequest(
					cl.AdminDriveCredentialsGetWithResponse(adminCtx, admin),
				)(t, http.StatusOK)

				r.NotNil(status.JSON200)
				r.False(status.JSON200.Configured)
				r.Equal(openapi.DriveCredentialsSource("none"), status.JSON200.Source)
			})

			t.Run("upload_status_delete_roundtrip", func(t *testing.T) {
				r := require.New(t)

				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_002_Frigg)
				admin := sh.WithSession(adminCtx)

				key := fakeServiceAccountKey(t, "storyden-test@example-project.iam.gserviceaccount.com")

				uploaded := tests.AssertRequest(
					cl.AdminDriveCredentialsUploadWithBodyWithResponse(adminCtx, "application/octet-stream", bytes.NewReader(key), admin),
				)(t, http.StatusOK)

				r.NotNil(uploaded.JSON200)
				r.True(uploaded.JSON200.Configured)
				r.Equal(openapi.DriveCredentialsSource("upload"), uploaded.JSON200.Source)
				r.NotNil(uploaded.JSON200.ServiceAccountEmail)
				r.Equal("storyden-test@example-project.iam.gserviceaccount.com", *uploaded.JSON200.ServiceAccountEmail)

				status := tests.AssertRequest(
					cl.AdminDriveCredentialsGetWithResponse(adminCtx, admin),
				)(t, http.StatusOK)

				r.NotNil(status.JSON200)
				r.True(status.JSON200.Configured)
				r.Equal(openapi.DriveCredentialsSource("upload"), status.JSON200.Source)

				deleted := tests.AssertRequest(
					cl.AdminDriveCredentialsDeleteWithResponse(adminCtx, admin),
				)(t, http.StatusOK)

				r.NotNil(deleted.JSON200)
				r.False(deleted.JSON200.Configured)
				r.Equal(openapi.DriveCredentialsSource("none"), deleted.JSON200.Source)
			})

			t.Run("rejects_invalid_json", func(t *testing.T) {
				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				admin := sh.WithSession(adminCtx)

				tests.AssertRequest(
					cl.AdminDriveCredentialsUploadWithBodyWithResponse(adminCtx, "application/octet-stream", bytes.NewReader([]byte("not json")), admin),
				)(t, http.StatusBadRequest)
			})

			t.Run("rejects_non_service_account_json", func(t *testing.T) {
				adminCtx, _ := e2e.WithAccount(root, aw, seed.Account_001_Odin)
				admin := sh.WithSession(adminCtx)

				tests.AssertRequest(
					cl.AdminDriveCredentialsUploadWithBodyWithResponse(adminCtx, "application/octet-stream", bytes.NewReader([]byte(`{"type":"authorized_user"}`)), admin),
				)(t, http.StatusBadRequest)
			})

			t.Run("denies_non_admin", func(t *testing.T) {
				memberCtx, _ := e2e.WithAccount(root, aw, seed.Account_003_Baldur)
				member := sh.WithSession(memberCtx)

				tests.AssertRequest(
					cl.AdminDriveCredentialsGetWithResponse(memberCtx, member),
				)(t, http.StatusForbidden)

				key := fakeServiceAccountKey(t, "member-blocked@example-project.iam.gserviceaccount.com")

				tests.AssertRequest(
					cl.AdminDriveCredentialsUploadWithBodyWithResponse(memberCtx, "application/octet-stream", bytes.NewReader(key), member),
				)(t, http.StatusForbidden)
			})

			t.Run("denies_anonymous", func(t *testing.T) {
				tests.AssertRequest(
					cl.AdminDriveCredentialsGetWithResponse(root),
				)(t, http.StatusUnauthorized)
			})
		}))
	}))
}
