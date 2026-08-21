package ai

import (
	"context"
	"encoding/json"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/invopop/jsonschema"
)

// StructuredPrompter is implemented by providers that can be constrained to
// emit JSON matching a schema. Keeping this an interface rather than a type
// switch means a test can substitute a fake, which is what makes the callers
// of PromptObject testable at all.
type StructuredPrompter interface {
	PromptObjectJSON(ctx context.Context, description, input string, schema any) (string, error)
}

// Usage counts tokens consumed since a provider was constructed.
type Usage struct {
	InputTokens  int
	OutputTokens int
}

func (u Usage) Total() int { return u.InputTokens + u.OutputTokens }

// UsageReporter is implemented by providers that track token consumption, so a
// batch job can report what a run actually cost.
type UsageReporter interface {
	Usage() Usage
}

func PromptObject[T any](ctx context.Context, prompter Prompter, description, input string, schema T) (*T, error) {
	structured, ok := prompter.(StructuredPrompter)
	if !ok {
		return nil, fault.Wrap(ErrStructuredUnsupported, fctx.With(ctx))
	}

	serialisedSchema, err := schemaFromObjectInstance(schema)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	payload, err := structured.PromptObjectJSON(ctx, description, input, serialisedSchema)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if payload == "" {
		return nil, fault.New("result json is empty", fctx.With(ctx))
	}

	var result T
	if err := json.Unmarshal([]byte(payload), &result); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &result, nil
}

func schemaFromObjectInstance[T any](instance T) (any, error) {
	r := jsonschema.Reflector{
		Anonymous:                 true,
		ExpandedStruct:            true,
		AllowAdditionalProperties: false,
		DoNotReference:            true,
	}

	schema := r.Reflect(instance)

	return schema, nil
}
