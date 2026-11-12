package ai

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
	"github.com/philippgille/chromem-go"

	"github.com/Southclaws/storyden/internal/config"
)

const (
	defaultChatModel      = openai.ChatModelGPT4_1
	defaultEmbeddingModel = string(chromem.EmbeddingModelOpenAI3Large)
)

type OpenAI struct {
	client *openai.Client
	model  string
	ef     func(ctx context.Context, text string) ([]float32, error)
}

func newOpenAI(cfg config.Config) (*OpenAI, error) {
	model := cfg.LanguageModelChatModel
	if model == "" {
		model = defaultChatModel
	}

	embeddingModel := cfg.LanguageModelEmbeddingModel
	if embeddingModel == "" {
		embeddingModel = defaultEmbeddingModel
	}

	key := cfg.OpenAIKey
	opts := []option.RequestOption{}

	var ef chromem.EmbeddingFunc
	if cfg.OpenAIBaseURL != "" {
		if key == "" {
			key = "ollama"
		}
		opts = append(opts, option.WithBaseURL(cfg.OpenAIBaseURL))
		ef = chromem.NewEmbeddingFuncOpenAICompat(cfg.OpenAIBaseURL, key, embeddingModel, nil)
	} else {
		ef = chromem.NewEmbeddingFuncOpenAI(key, chromem.EmbeddingModelOpenAI(embeddingModel))
	}

	opts = append(opts, option.WithAPIKey(key))
	client := openai.NewClient(opts...)

	return &OpenAI{client: &client, model: model, ef: ef}, nil
}

func (o *OpenAI) Prompt(ctx context.Context, input string) (*Result, error) {
	res, err := o.client.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model: o.model,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.UserMessage(input),
		},
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if len(res.Choices) == 0 {
		return nil, fault.New("result is empty")
	}

	return &Result{
		Answer: res.Choices[0].Message.Content,
	}, nil
}

func (o *OpenAI) PromptStream(ctx context.Context, input string) (func(yield func(string, error) bool), error) {
	iter := func(yield func(string, error) bool) {
		stream := o.client.Chat.Completions.NewStreaming(ctx, openai.ChatCompletionNewParams{
			Model: o.model,
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage(input),
			},
		})

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if !stream.Next() {
				break
			}

			chunk := stream.Current()

			if len(chunk.Choices) > 0 {
				if !yield(chunk.Choices[0].Delta.Content, nil) {
					return
				}
			}
		}

		if err := stream.Err(); err != nil {
			yield("", fault.Wrap(err, fctx.With(ctx)))
			return
		}
	}

	return iter, nil
}

func (o *OpenAI) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return o.ef
}
