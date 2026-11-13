package ai

import (
	"context"

	"github.com/Southclaws/storyden/internal/config"
)

type Result struct {
	Answer string
}

type Embedder func(ctx context.Context, text string) ([]float32, error)

type Prompter interface {
	Prompt(ctx context.Context, input string, opts ...Option) (*Result, error)
	PromptStream(ctx context.Context, input string, opts ...Option) (func(yield func(string, error) bool), error)
	EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error)
}

func New(cfg config.Config) (Prompter, error) {
	switch cfg.LanguageModelProvider {
	case "openai":
		p, err := newOpenAI(cfg)
		if err != nil {
			return nil, err
		}
		return NewQueue(p), nil

	case "mock":
		p, err := newMock()
		if err != nil {
			return nil, err
		}
		return NewQueue(p), nil

	default:
		return &Disabled{}, nil
	}
}

type Disabled struct{}

func (d *Disabled) Prompt(ctx context.Context, input string, opts ...Option) (*Result, error) {
	return nil, nil
}

func (d *Disabled) PromptStream(ctx context.Context, input string, opts ...Option) (func(yield func(string, error) bool), error) {
	return nil, nil
}

func (d *Disabled) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return nil
}

func IsOpenAI(p Prompter) bool {
	if q, ok := p.(*Queue); ok {
		p = q.inner
	}
	_, ok := p.(*OpenAI)
	return ok
}

func IsDisabled(p Prompter) bool {
	_, ok := p.(*Disabled)
	return ok
}
