package ai

import (
	"container/list"
	"context"
	"crypto/sha256"
	"slices"
	"sync"
)

const embedCacheSize = 500

type embedFunc = func(ctx context.Context, text string) ([]float32, error)

type embedEntry struct {
	key [32]byte
	vec []float32
}

type embedCache struct {
	mu    sync.Mutex
	size  int
	order *list.List
	items map[[32]byte]*list.Element
}

func newEmbedCache(size int) *embedCache {
	return &embedCache{size: size, order: list.New(), items: map[[32]byte]*list.Element{}}
}

func (c *embedCache) get(key [32]byte) ([]float32, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	el, ok := c.items[key]
	if !ok {
		return nil, false
	}
	c.order.MoveToFront(el)
	return slices.Clone(el.Value.(*embedEntry).vec), true
}

func (c *embedCache) put(key [32]byte, vec []float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if el, ok := c.items[key]; ok {
		c.order.MoveToFront(el)
		return
	}
	c.items[key] = c.order.PushFront(&embedEntry{key: key, vec: slices.Clone(vec)})
	for c.order.Len() > c.size {
		last := c.order.Back()
		c.order.Remove(last)
		delete(c.items, last.Value.(*embedEntry).key)
	}
}

// embedder caches embeddings and runs one at a time. Background embeddings
// additionally wait until no member is waiting for a chat answer.
func (q *Queue) embedder(inner embedFunc) embedFunc {
	return func(ctx context.Context, text string) ([]float32, error) {
		key := sha256.Sum256([]byte(text))
		if v, ok := q.embeds.get(key); ok {
			return v, nil
		}

		if isLowPriority(ctx) {
			if err := q.waitIdle(ctx); err != nil {
				return nil, err
			}
		}

		select {
		case q.embedSlot <- struct{}{}:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		defer func() { <-q.embedSlot }()

		if v, ok := q.embeds.get(key); ok {
			return v, nil
		}

		v, err := inner(ctx, text)
		if err != nil {
			return nil, err
		}
		q.embeds.put(key, v)
		return v, nil
	}
}

func (q *Queue) waitIdle(ctx context.Context) error {
	for {
		q.mu.Lock()
		idle := q.highWaiting == 0 && (!q.busy || q.runningLow != nil)
		wake := q.wake
		q.mu.Unlock()
		if idle {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-wake:
		}
	}
}
