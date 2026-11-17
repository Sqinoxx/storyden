package ai

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingEmbedder struct {
	*blockingPrompter
	calls atomic.Int32
}

func (c *countingEmbedder) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return func(ctx context.Context, text string) ([]float32, error) {
		c.calls.Add(1)
		return []float32{float32(len(text)), 1}, nil
	}
}

func TestEmbeddingsAreCached(t *testing.T) {
	c := &countingEmbedder{blockingPrompter: newBlocking()}
	embed := newTestQueue(c).EmbeddingFunc()

	a, err := embed(context.Background(), "Zahnmedizin")
	require.NoError(t, err)
	a[0] = 99

	b, err := embed(context.Background(), "Zahnmedizin")
	require.NoError(t, err)
	assert.Equal(t, float32(11), b[0])
	assert.Equal(t, int32(1), c.calls.Load())

	_, err = embed(context.Background(), "Prothetik")
	require.NoError(t, err)
	assert.Equal(t, int32(2), c.calls.Load())
}

func TestEmbedCacheEvictsOldest(t *testing.T) {
	c := newEmbedCache(2)
	c.put([32]byte{1}, []float32{1})
	c.put([32]byte{2}, []float32{2})
	_, _ = c.get([32]byte{1})
	c.put([32]byte{3}, []float32{3})

	_, ok := c.get([32]byte{2})
	assert.False(t, ok)
	_, ok = c.get([32]byte{1})
	assert.True(t, ok)
}

func TestBackgroundEmbeddingWaitsForMemberJob(t *testing.T) {
	c := &countingEmbedder{blockingPrompter: newBlocking()}
	q := newTestQueue(c)
	embed := q.EmbeddingFunc()

	go q.Prompt(context.Background(), "member")
	<-c.started

	done := make(chan error, 1)
	go func() {
		_, err := embed(LowPriorityContext(context.Background()), "background")
		done <- err
	}()

	select {
	case <-done:
		t.Fatal("background embedding ran while a member job was running")
	case <-time.After(50 * time.Millisecond):
	}

	_, err := embed(context.Background(), "member embedding")
	require.NoError(t, err)

	c.release <- struct{}{}
	require.NoError(t, <-done)
	assert.Equal(t, int32(2), c.calls.Load())
}

func TestDisabledEmbeddingStaysNil(t *testing.T) {
	assert.Nil(t, newTestQueue(newBlocking()).EmbeddingFunc())
}
