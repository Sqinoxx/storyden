package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genai"
)

type classificationFixture struct {
	Titel string   `json:"titel" jsonschema:"title=Titel,description=Ein lesbarer Titel"`
	Jahr  string   `json:"jahr" jsonschema:"title=Jahr,description=Vierstellige Jahreszahl"`
	Tags  []string `json:"tags" jsonschema:"title=Tags,description=Bis zu vier Tags,items=string"`
}

// invopop/jsonschema always emits $schema, and Gemini rejects the whole request
// with 400 INVALID_ARGUMENT when it sees it. This is the single thing standing
// between a working structured call and every request failing.
func TestSanitiseSchemaStripsSchemaKeywordFromRealReflectedSchema(t *testing.T) {
	t.Parallel()

	reflected, err := schemaFromObjectInstance(classificationFixture{})
	require.NoError(t, err)

	raw, err := json.Marshal(reflected)
	require.NoError(t, err)
	require.Contains(t, string(raw), "$schema", "precondition: the reflector should emit $schema")

	sanitised, err := sanitiseSchemaForGemini(reflected)
	require.NoError(t, err)

	out, err := json.Marshal(sanitised)
	require.NoError(t, err)

	assert.NotContains(t, string(out), "$schema")

	// The parts Gemini actually needs must survive intact.
	assert.Contains(t, string(out), "titel")
	assert.Contains(t, string(out), "properties")
	assert.Contains(t, string(out), "required")
	assert.Contains(t, string(out), "additionalProperties")
}

func TestSanitiseSchemaStripsNestedRejectedKeys(t *testing.T) {
	t.Parallel()

	input := map[string]any{
		"$schema":              "https://json-schema.org/draft/2020-12/schema",
		"$id":                  "https://example.test/root",
		"type":                 "object",
		"additionalProperties": false,
		"required":             []any{"outer"},
		"properties": map[string]any{
			"outer": map[string]any{
				"$comment": "should go",
				"type":     "object",
				"properties": map[string]any{
					"inner": map[string]any{
						"$anchor": "should go too",
						"type":    "string",
						"enum":    []any{"a", "b"},
					},
				},
			},
			"list": map[string]any{
				"type": "array",
				"items": map[string]any{
					"$schema": "should go",
					"type":    "string",
				},
			},
			"choice": map[string]any{
				"anyOf": []any{
					map[string]any{"$id": "should go", "type": "string"},
					map[string]any{"type": "number"},
				},
			},
		},
	}

	sanitised, err := sanitiseSchemaForGemini(input)
	require.NoError(t, err)

	encoded, err := json.Marshal(sanitised)
	require.NoError(t, err)

	for _, rejected := range []string{"$schema", "$id", "$comment", "$anchor"} {
		assert.NotContains(t, string(encoded), rejected, "%q should have been stripped at every level", rejected)
	}

	var round map[string]any
	require.NoError(t, json.Unmarshal(encoded, &round))

	props := round["properties"].(map[string]any)
	inner := props["outer"].(map[string]any)["properties"].(map[string]any)["inner"].(map[string]any)
	assert.Equal(t, "string", inner["type"])
	assert.Equal(t, []any{"a", "b"}, inner["enum"], "enum constraints must survive")

	items := props["list"].(map[string]any)["items"].(map[string]any)
	assert.Equal(t, "string", items["type"])

	choices := props["choice"].(map[string]any)["anyOf"].([]any)
	assert.Len(t, choices, 2, "anyOf branches must survive")
	assert.Equal(t, false, round["additionalProperties"])
}

func TestClassifyGoogleError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		code int
		want error
	}{
		{name: "quota exhausted is rate limited", code: http.StatusTooManyRequests, want: ErrRateLimited},
		{name: "internal error is transient", code: http.StatusInternalServerError, want: ErrProviderUnavailable},
		{name: "unavailable is transient", code: http.StatusServiceUnavailable, want: ErrProviderUnavailable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// genai returns APIError by value, so a value target is required.
			got := classifyGoogleError(context.Background(), genai.APIError{Code: tt.code, Message: "boom"})

			assert.True(t, errors.Is(got, tt.want), "expected %v, got %v", tt.want, got)
			assert.True(t, isRetryable(got))
		})
	}
}

// A rejected schema is a permanent failure — retrying it just burns quota.
func TestClassifyGoogleErrorTreatsBadRequestAsPermanent(t *testing.T) {
	t.Parallel()

	got := classifyGoogleError(context.Background(), genai.APIError{
		Code:    http.StatusBadRequest,
		Status:  "INVALID_ARGUMENT",
		Message: `Unknown name "$schema"`,
	})

	assert.False(t, errors.Is(got, ErrRateLimited))
	assert.False(t, errors.Is(got, ErrProviderUnavailable))
	assert.False(t, isRetryable(got))
}

func TestClassifyGoogleErrorPassesThroughNonAPIErrors(t *testing.T) {
	t.Parallel()

	sentinel := errors.New("dial tcp: connection refused")

	got := classifyGoogleError(context.Background(), sentinel)

	assert.True(t, errors.Is(got, sentinel))
	assert.False(t, isRetryable(got))
	assert.Nil(t, classifyGoogleError(context.Background(), nil))
}

func TestRPMLimiterPacesRequests(t *testing.T) {
	t.Parallel()

	// 600 per minute is one every 100ms; the first is immediate, so two waits
	// must take at least one interval in total.
	limiter := newRPMLimiter(600)
	ctx := context.Background()

	started := time.Now()
	require.NoError(t, limiter.Wait(ctx))
	require.NoError(t, limiter.Wait(ctx))

	assert.GreaterOrEqual(t, time.Since(started), 90*time.Millisecond)
}

func TestRPMLimiterZeroDisablesPacing(t *testing.T) {
	t.Parallel()

	limiter := newRPMLimiter(0)

	started := time.Now()
	for range 50 {
		require.NoError(t, limiter.Wait(context.Background()))
	}

	assert.Less(t, time.Since(started), time.Second)
}

// A long wait must not outlive its context, otherwise ctrl-c on a paced batch
// run would hang for minutes.
func TestRPMLimiterRespectsContextCancellation(t *testing.T) {
	t.Parallel()

	limiter := newRPMLimiter(1)
	require.NoError(t, limiter.Wait(context.Background()))

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	assert.Error(t, limiter.Wait(ctx))
}
