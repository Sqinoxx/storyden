package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"golang.org/x/time/rate"
	"google.golang.org/genai"

	"github.com/Southclaws/storyden/internal/config"
)

// Google implements Prompter against the Gemini Developer API. It needs an API
// key from Google AI Studio (aistudio.google.com/apikey) — a Google AI
// Pro/Ultra subscription is a consumer chat product and does not grant API
// access on its own.
type Google struct {
	client         *genai.Client
	model          string
	embeddingModel string
	limiter        *rate.Limiter
	maxRetries     int

	mu    sync.Mutex
	usage Usage
}

func newGoogle(ctx context.Context, cfg config.Config) (*Google, error) {
	if cfg.GoogleAIKey == "" {
		return nil, fault.Wrap(ErrNotConfigured,
			fctx.With(ctx),
			fmsg.WithDesc("missing google ai key", "Set GOOGLE_AI_API_KEY to an API key from https://aistudio.google.com/apikey."),
		)
	}

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
		limiter:        newRPMLimiter(cfg.GoogleAIMaxRPM),
		maxRetries:     cfg.GoogleAIMaxRetries,
	}, nil
}

// newRPMLimiter paces requests to stay inside a requests-per-minute quota. A
// burst of one keeps the spacing strictly even, which matters on the free tier
// where a burst immediately trips the per-minute limit.
func newRPMLimiter(rpm int) *rate.Limiter {
	if rpm <= 0 {
		return rate.NewLimiter(rate.Inf, 1)
	}

	return rate.NewLimiter(rate.Every(time.Minute/time.Duration(rpm)), 1)
}

func (g *Google) Prompt(ctx context.Context, input string) (*Result, error) {
	res, err := g.generate(ctx, []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}, nil)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	text := res.Text()
	if text == "" {
		return nil, fault.New("result is empty", fctx.With(ctx))
	}

	return &Result{Answer: text}, nil
}

func (g *Google) PromptStream(ctx context.Context, input string) (func(yield func(string, error) bool), error) {
	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

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
				yield("", classifyGoogleError(ctx, err))
				return
			}

			g.recordUsage(chunk)

			if text := chunk.Text(); text != "" {
				if !yield(text, nil) {
					return
				}
			}
		}
	}

	return iter, nil
}

func (g *Google) PromptObjectJSON(ctx context.Context, description, input string, schema any) (string, error) {
	sanitised, err := sanitiseSchemaForGemini(schema)
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	res, err := g.generate(ctx, []*genai.Content{
		genai.NewContentFromText(input, genai.RoleUser),
	}, &genai.GenerateContentConfig{
		ResponseMIMEType:   "application/json",
		ResponseJsonSchema: sanitised,
		SystemInstruction:  genai.NewContentFromText(description, genai.RoleUser),
	})
	if err != nil {
		return "", fault.Wrap(err, fctx.With(ctx))
	}

	return res.Text(), nil
}

func (g *Google) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return func(ctx context.Context, text string) ([]float32, error) {
		if err := g.limiter.Wait(ctx); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		res, err := g.client.Models.EmbedContent(ctx, g.embeddingModel, []*genai.Content{
			genai.NewContentFromText(text, genai.RoleUser),
		}, nil)
		if err != nil {
			return nil, classifyGoogleError(ctx, err)
		}

		if len(res.Embeddings) == 0 {
			return nil, fault.New("no embeddings returned", fctx.With(ctx))
		}

		return res.Embeddings[0].Values, nil
	}
}

func (g *Google) Usage() Usage {
	g.mu.Lock()
	defer g.mu.Unlock()

	return g.usage
}

// generate paces, calls and retries. The genai SDK only retries file uploads
// internally, so transient failures on GenerateContent have to be handled here.
func (g *Google) generate(ctx context.Context, contents []*genai.Content, cfg *genai.GenerateContentConfig) (*genai.GenerateContentResponse, error) {
	attempts := g.maxRetries
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error

	for attempt := range attempts {
		if err := g.limiter.Wait(ctx); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}

		res, err := g.client.Models.GenerateContent(ctx, g.model, contents, cfg)
		if err == nil {
			g.recordUsage(res)
			return res, nil
		}

		lastErr = classifyGoogleError(ctx, err)

		if !isRetryable(lastErr) {
			return nil, lastErr
		}

		if attempt == attempts-1 {
			break
		}

		if err := sleepWithBackoff(ctx, attempt); err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx))
		}
	}

	return nil, lastErr
}

func (g *Google) recordUsage(res *genai.GenerateContentResponse) {
	if res == nil || res.UsageMetadata == nil {
		return
	}

	g.mu.Lock()
	defer g.mu.Unlock()

	g.usage.InputTokens += int(res.UsageMetadata.PromptTokenCount)
	g.usage.OutputTokens += int(res.UsageMetadata.CandidatesTokenCount)
}

func isRetryable(err error) bool {
	return errors.Is(err, ErrRateLimited) || errors.Is(err, ErrProviderUnavailable)
}

func sleepWithBackoff(ctx context.Context, attempt int) error {
	delay := time.Second * time.Duration(1<<attempt)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(delay):
		return nil
	}
}

// classifyGoogleError maps the API's HTTP status onto the package sentinels.
// genai returns APIError by value, so errors.As needs a value target.
func classifyGoogleError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	var apiErr genai.APIError
	if !errors.As(err, &apiErr) {
		return fault.Wrap(err, fctx.With(ctx))
	}

	switch {
	case apiErr.Code == http.StatusTooManyRequests:
		return fault.Wrap(ErrRateLimited, fctx.With(ctx), fmsg.WithDesc("quota exceeded", apiErr.Message))

	case apiErr.Code >= 500:
		return fault.Wrap(ErrProviderUnavailable, fctx.With(ctx), fmsg.WithDesc(apiErr.Status, apiErr.Message))

	default:
		return fault.Wrap(err, fctx.With(ctx), fmsg.WithDesc(apiErr.Status, apiErr.Message))
	}
}

// geminiRejectedSchemaKeys are JSON Schema keywords the Gemini API refuses.
// invopop/jsonschema always emits $schema, which alone is enough to fail every
// structured request with 400 INVALID_ARGUMENT.
var geminiRejectedSchemaKeys = []string{"$schema", "$id", "$comment", "$anchor"}

// sanitiseSchemaForGemini round-trips the reflected schema through JSON and
// strips the keywords Gemini rejects, at every level of nesting.
func sanitiseSchemaForGemini(schema any) (any, error) {
	encoded, err := json.Marshal(schema)
	if err != nil {
		return nil, fault.Wrap(err)
	}

	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		return nil, fault.Wrap(err)
	}

	return stripSchemaKeys(decoded), nil
}

func stripSchemaKeys(node any) any {
	switch typed := node.(type) {
	case map[string]any:
		for _, key := range geminiRejectedSchemaKeys {
			delete(typed, key)
		}

		for key, value := range typed {
			typed[key] = stripSchemaKeys(value)
		}

		return typed

	case []any:
		for i, value := range typed {
			typed[i] = stripSchemaKeys(value)
		}

		return typed

	default:
		return node
	}
}
