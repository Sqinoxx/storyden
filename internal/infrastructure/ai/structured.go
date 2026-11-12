package ai

import (
	"context"
	"encoding/json"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/invopop/jsonschema"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/packages/param"
	"github.com/openai/openai-go/shared"
)

func PromptObject[T any](ctx context.Context, prompter Prompter, description, input string, schema T) (*T, error) {
	serialisedSchema, err := schemaFromObjectInstance(schema)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	var result T
	if err := PromptSchema(ctx, prompter, description, input, serialisedSchema, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// PromptSchema is PromptObject for schemas built at runtime, such as one that
// constrains a field to an enum of values loaded from the database.
func PromptSchema(ctx context.Context, prompter Prompter, description, input string, schema any, out any) error {
	s, ok := prompter.(*OpenAI)
	if !ok {
		return fault.New("structured prompt only supported with OpenAI prompter")
	}

	res, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:       s.model,
		Temperature: param.NewOpt(0.0),
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        "json_schema",
					Strict:      param.NewOpt(true),
					Description: param.NewOpt(description),
					Schema:      schema,
				},
			},
		},
	})
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	if len(res.Choices) == 0 {
		return fault.New("result choices are empty")
	}

	choice := res.Choices[0]

	if choice.Message.JSON.Content.Raw() == "" {
		return fault.New("result json is empty")
	}

	if !choice.Message.JSON.Content.Valid() {
		return fault.New("result is not valid JSON")
	}

	if err := json.Unmarshal([]byte(choice.Message.Content), out); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
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
