package ai

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blockingPrompter struct {
	running atomic.Int32
	max     atomic.Int32
	release chan struct{}
	started chan struct{}
}

func newBlocking() *blockingPrompter {
	return &blockingPrompter{release: make(chan struct{}), started: make(chan struct{}, 16)}
}

func (b *blockingPrompter) Prompt(ctx context.Context, input string, opts ...Option) (*Result, error) {
	n := b.running.Add(1)
	defer b.running.Add(-1)
	for {
		m := b.max.Load()
		if n <= m || b.max.CompareAndSwap(m, n) {
			break
		}
	}
	b.started <- struct{}{}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-b.release:
		return &Result{Answer: input}, nil
	}
}

func (b *blockingPrompter) PromptStream(ctx context.Context, input string, opts ...Option) (func(yield func(string, error) bool), error) {
	return func(yield func(string, error) bool) {
		r, err := b.Prompt(ctx, input, opts...)
		if err != nil {
			yield("", err)
			return
		}
		yield(r.Answer, nil)
	}, nil
}

func (b *blockingPrompter) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return nil
}

func newTestQueue(inner Prompter) *Queue {
	q := NewQueue(inner)
	q.idleGap = 20 * time.Millisecond
	return q
}

func TestQueueRunsOneJobAtATime(t *testing.T) {
	b := newBlocking()
	q := newTestQueue(b)

	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := q.Prompt(context.Background(), "x")
			assert.NoError(t, err)
		}()
	}
	for range 3 {
		<-b.started
		b.release <- struct{}{}
	}
	wg.Wait()
	assert.Equal(t, int32(1), b.max.Load())
}

func TestQueueRejectsWhenFull(t *testing.T) {
	b := newBlocking()
	q := newTestQueue(b)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go q.Prompt(ctx, "running")
	<-b.started
	for range defaultMaxHighWaiting {
		go q.Prompt(ctx, "waiting")
	}
	require.Eventually(t, func() bool {
		q.mu.Lock()
		defer q.mu.Unlock()
		return q.highWaiting == defaultMaxHighWaiting
	}, time.Second, time.Millisecond)

	_, err := q.Prompt(ctx, "overflow")
	assert.ErrorIs(t, err, ErrBusy)
}

func TestQueueHighPreemptsLow(t *testing.T) {
	b := newBlocking()
	q := newTestQueue(b)

	lowErr := make(chan error, 1)
	go func() {
		_, err := q.Prompt(context.Background(), "low", WithLowPriority())
		lowErr <- err
	}()
	<-b.started

	highDone := make(chan error, 1)
	go func() {
		_, err := q.Prompt(context.Background(), "high")
		highDone <- err
	}()

	assert.True(t, errors.Is(<-lowErr, ErrPreempted))
	<-b.started
	b.release <- struct{}{}
	assert.NoError(t, <-highDone)
}

func TestQueueLowWaitsForIdleGap(t *testing.T) {
	b := newBlocking()
	q := newTestQueue(b)
	q.idleGap = 150 * time.Millisecond

	go func() {
		<-b.started
		b.release <- struct{}{}
	}()
	_, err := q.Prompt(context.Background(), "high")
	require.NoError(t, err)

	start := time.Now()
	go func() {
		<-b.started
		b.release <- struct{}{}
	}()
	_, err = q.Prompt(context.Background(), "low", WithLowPriority())
	require.NoError(t, err)
	assert.GreaterOrEqual(t, time.Since(start), 100*time.Millisecond)
}

func TestQueueStreamReleasesSlot(t *testing.T) {
	b := newBlocking()
	q := newTestQueue(b)

	go func() {
		for range 2 {
			<-b.started
			b.release <- struct{}{}
		}
	}()

	for range 2 {
		it, err := q.PromptStream(context.Background(), "s")
		require.NoError(t, err)
		var out string
		for chunk, err := range it {
			require.NoError(t, err)
			out += chunk
		}
		assert.Equal(t, "s", out)
	}
}
