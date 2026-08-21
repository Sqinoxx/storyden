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
	"google.golang.org/genai"
)

func PromptObject[T any](ctx context.Context, prompter Prompter, description, input string, schema T) (*T, error) {
	switch p := prompter.(type) {
	case *OpenAI:
		return promptObjectOpenAI(ctx, p, description, input, schema)
	case *Google:
		return promptObjectGoogle(ctx, p, description, input, schema)
	default:
		return nil, fault.New("structured prompt only supported with OpenAI or Google prompters")
	}
}

func promptObjectOpenAI[T any](ctx context.Context, s *OpenAI, description, input string, schema T) (*T, error) {
	serialisedSchema, err := schemaFromObjectInstance(schema)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	res, err := s.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: openai.ChatModelGPT4_1,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &shared.ResponseFormatJSONSchemaParam{
				JSONSchema: shared.ResponseFormatJSONSchemaJSONSchemaParam{
					Name:        "json_schema",
					Strict:      param.NewOpt(true),
					Description: param.NewOpt(description),
					Schema:      serialisedSchema,
				},
			},
		},
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if len(res.Choices) == 0 {
		return nil, fault.New("result choices are empty")
	}

	if res.Choices[0].Message.JSON.Content.Raw() == "" {
		return nil, fault.New("result json is empty")
	}

	choice := res.Choices[0]

	if choice.Message.JSON.Content.Valid() == false {
		// TODO: Retry a few times with a backoff?
		return nil, fault.New("result is not valid JSON")
	}

	payload := choice.Message.Content

	var result T
	err = json.Unmarshal([]byte(payload), &result)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &result, nil
}

func promptObjectGoogle[T any](ctx context.Context, g *Google, description, input string, schema T) (*T, error) {
	serialisedSchema, err := schemaFromObjectInstance(schema)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	res, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}, &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: serialisedSchema,
		SystemInstruction:  genai.NewContentFromText(description, genai.RoleUser),
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	payload := res.Text()
	if payload == "" {
		return nil, fault.New("result json is empty")
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
