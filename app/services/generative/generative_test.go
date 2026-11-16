package generative

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

type capture struct {
	input string
	opts  ai.Options
}

func (c *capture) Prompt(ctx context.Context, input string, opts ...ai.Option) (*ai.Result, error) {
	c.input = input
	c.opts = ai.ResolveOptions(opts...)
	return &ai.Result{Answer: "Erster Titel\n\n Zweiter Titel "}, nil
}

func (c *capture) PromptStream(ctx context.Context, input string, opts ...ai.Option) (func(yield func(string, error) bool), error) {
	return nil, nil
}

func (c *capture) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return nil
}

func longContent(t *testing.T) datagraph.Content {
	c, err := datagraph.NewRichText("<p>" + strings.Repeat("Wurzelkanal ", 2000) + "</p>")
	require.NoError(t, err)
	return c
}

func TestSuggestTitleIsFastAndBounded(t *testing.T) {
	c := &capture{}
	titles, err := newGenerator(c).SuggestTitle(context.Background(), longContent(t))
	require.NoError(t, err)

	assert.Equal(t, []string{"Erster Titel", "Zweiter Titel"}, titles)
	assert.True(t, c.opts.Fast)
	assert.Equal(t, titleMaxTokens, c.opts.MaxTokens)
	assert.Less(t, len(c.input), maxInputChars+1000)
}

func TestSummariseIsFastAndBounded(t *testing.T) {
	c := &capture{}
	_, err := newGenerator(c).Summarise(context.Background(), longContent(t))
	require.NoError(t, err)

	assert.True(t, c.opts.Fast)
	assert.Equal(t, summaryMaxTokens, c.opts.MaxTokens)
	assert.Less(t, len(c.input), maxInputChars+2500)
}
