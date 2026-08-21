package ai

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

type SuggestTagsResultSchema struct {
	Tags []string `json:"tags" jsonschema:"title=Tags,description=List of suggested tags,items=string"`
}

func Test_schemaFromObjectInstance(t *testing.T) {
	r := require.New(t)

	sc, err := schemaFromObjectInstance(SuggestTagsResultSchema{})
	r.NoError(err)

	j, _ := json.MarshalIndent(sc, "", "  ")

	r.Equal(`{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "properties": {
    "tags": {
      "items": {
        "type": "string"
      },
      "type": "array",
      "title": "Tags",
      "description": "List of suggested tags"
    }
  },
  "additionalProperties": false,
  "type": "object",
  "required": [
    "tags"
  ]
}`, string(j))
}

type recordingPrompter struct {
	Mock
	gotSchema      any
	gotDescription string
	reply          string
	err            error
}

func (r *recordingPrompter) PromptObjectJSON(ctx context.Context, description, input string, schema any) (string, error) {
	r.gotSchema = schema
	r.gotDescription = description
	return r.reply, r.err
}

func TestPromptObjectUnmarshalsProviderJSON(t *testing.T) {
	r := require.New(t)

	prompter := &recordingPrompter{reply: `{"tags":["kfo","mkg"]}`}

	out, err := PromptObject(context.Background(), prompter, "classify", "input", SuggestTagsResultSchema{})
	r.NoError(err)
	r.Equal([]string{"kfo", "mkg"}, out.Tags)
	r.Equal("classify", prompter.gotDescription)
	r.NotNil(prompter.gotSchema, "the reflected schema must reach the provider")
}

// A provider that cannot do structured output must say so plainly rather than
// failing later with a confusing unmarshal error.
func TestPromptObjectRejectsProviderWithoutStructuredSupport(t *testing.T) {
	r := require.New(t)

	_, err := PromptObject(context.Background(), &Disabled{}, "classify", "input", SuggestTagsResultSchema{})

	r.Error(err)
	r.True(errors.Is(err, ErrStructuredUnsupported))
}

func TestPromptObjectPropagatesProviderError(t *testing.T) {
	r := require.New(t)

	_, err := PromptObject(context.Background(), &recordingPrompter{err: ErrRateLimited}, "classify", "input", SuggestTagsResultSchema{})

	r.Error(err)
	r.True(errors.Is(err, ErrRateLimited), "callers branch on this to stop a batch run")
}

func TestPromptObjectRejectsEmptyResult(t *testing.T) {
	r := require.New(t)

	_, err := PromptObject(context.Background(), &recordingPrompter{reply: ""}, "classify", "input", SuggestTagsResultSchema{})

	r.Error(err)
}

// The mock returns an empty object, which must unmarshal cleanly into any
// schema so the whole chain can run in tests without a network.
func TestPromptObjectWithMockYieldsZeroValues(t *testing.T) {
	r := require.New(t)

	out, err := PromptObject(context.Background(), &Mock{}, "classify", "input", SuggestTagsResultSchema{})

	r.NoError(err)
	r.Empty(out.Tags)
}
