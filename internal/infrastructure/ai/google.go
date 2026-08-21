package ai

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"google.golang.org/genai"

	"github.com/Southclaws/storyden/internal/config"
)

// Google implements Prompter against the Gemini Developer API. It needs an
// API key from Google AI Studio (aistudio.google.com/apikey) — a Google AI
// Pro/Ultra subscription is a consumer chat product and does not grant API
// access on its own.
type Google struct {
	client         *genai.Client
	model          string
	embeddingModel string
}

func newGoogle(ctx context.Context, cfg config.Config) (*Google, error) {
	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.GoogleAIKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return &Google{
		client:         client,
		model:          cfg.GoogleAIModel,
		embeddingModel: cfg.GoogleAIEmbeddingModel,
	}, nil
}

func (g *Google) Prompt(ctx context.Context, input string) (*Result, error) {
	res, err := g.client.Models.GenerateContent(ctx, g.model, []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}, nil)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	text := res.Text()
	if text == "" {
		return nil, fault.New("result is empty")
	}

	return &Result{Answer: text}, nil
}

func (g *Google) PromptStream(ctx context.Context, input string) (func(yield func(string, error) bool), error) {
	stream := g.client.Models.GenerateContentStream(ctx, g.model, []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}, nil)

	iter := func(yield func(string, error) bool) {
		for chunk, err := range stream {
			select {
			case <-ctx.Done():
				return
			default:
			}

			if err != nil {
				yield("", fault.Wrap(err, fctx.With(ctx)))
				return
			}

			if text := chunk.Text(); text != "" {
				if !yield(text, nil) {
					return
				}
			}
		}
	}

	return iter, nil
}

func (g *Google) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return func(ctx context.Context, text string) ([]float32, error) {
		res, err := g.client.Models.EmbedContent(ctx, g.embeddingModel, []*genai.Content{
			genai.NewContentFromText(text, genai.RoleUser),
		}, nil)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		if len(res.Embeddings) == 0 {
			return nil, fault.New("no embeddings returned")
		}

		return res.Embeddings[0].Values, nil
	}
}
