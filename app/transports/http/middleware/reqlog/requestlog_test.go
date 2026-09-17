package reqlog

import (
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedactedQueryHidesSensitiveValues covers B19: OAuth codes/state/tokens
// and password reset/verification tokens were logged in full as part of the
// request's query string on every single request, even though logs reach a
// wider audience (aggregators, retention, more people) than the app itself.
func TestRedactedQueryHidesSensitiveValues(t *testing.T) {
	t.Parallel()

	for _, key := range []string{"code", "state", "token", "user_code", "access_token", "refresh_token"} {
		t.Run(key, func(t *testing.T) {
			t.Parallel()

			u, err := url.Parse("/api/oauth/callback?" + key + "=super-secret-value&foo=bar")
			require.NoError(t, err)

			got := redactedQuery(u)

			values, err := url.ParseQuery(got)
			require.NoError(t, err)

			assert.Equal(t, redacted, values.Get(key), "sensitive key must be redacted")
			assert.Equal(t, "bar", values.Get("foo"), "unrelated keys must survive untouched")
			assert.NotContains(t, got, "super-secret-value")
		})
	}
}

func TestRedactedQueryPreservesRepeatedKeys(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("/x?token=one&token=two&tag=a&tag=b")
	require.NoError(t, err)

	got := redactedQuery(u)

	values, err := url.ParseQuery(got)
	require.NoError(t, err)

	require.Len(t, values["token"], 2)
	assert.Equal(t, []string{redacted, redacted}, values["token"])
	assert.Equal(t, []string{"a", "b"}, values["tag"])
}

func TestRedactedQueryLeavesOrdinaryParamsUntouched(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("/x?page=2&limit=10")
	require.NoError(t, err)

	assert.Equal(t, u.Query().Encode(), redactedQuery(u))
}

func TestRedactedQueryEmpty(t *testing.T) {
	t.Parallel()

	u, err := url.Parse("/x")
	require.NoError(t, err)

	assert.Equal(t, "", redactedQuery(u))
}
