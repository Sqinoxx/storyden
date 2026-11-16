// Package summary_stream fans one language model answer out to every caller
// waiting for the same summary.
package summary_stream

import (
	"context"
	"sync"
)

type Event struct {
	Text string
	HTML string
	Done bool
	Err  error
}

type Stream = func(yield func(Event) bool)

func Ready(html string) Stream {
	return func(yield func(Event) bool) {
		yield(Event{Done: true, HTML: html})
	}
}

type Broadcast struct {
	mu     sync.Mutex
	chunks []string
	done   bool
	html   string
	err    error
	wake   chan struct{}
}

func New() *Broadcast {
	return &Broadcast{wake: make(chan struct{})}
}

func (b *Broadcast) notify() {
	close(b.wake)
	b.wake = make(chan struct{})
}

func (b *Broadcast) Push(chunk string) {
	if chunk == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chunks = append(b.chunks, chunk)
	b.notify()
}

func (b *Broadcast) Finish(html string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.done = true
	b.html = html
	b.err = err
	b.notify()
}

func (b *Broadcast) Subscribe(ctx context.Context) Stream {
	return func(yield func(Event) bool) {
		next := 0
		for {
			b.mu.Lock()
			pending := append([]string{}, b.chunks[next:]...)
			next = len(b.chunks)
			done, html, err := b.done, b.html, b.err
			wake := b.wake
			b.mu.Unlock()

			for _, c := range pending {
				if !yield(Event{Text: c}) {
					return
				}
			}
			if done {
				if err != nil {
					yield(Event{Done: true, Err: err})
				} else {
					yield(Event{Done: true, HTML: html})
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			case <-wake:
			}
		}
	}
}

// Group shares one running Broadcast per key.
type Group[K comparable] struct {
	mu   sync.Mutex
	live map[K]*Broadcast
}

func NewGroup[K comparable]() *Group[K] {
	return &Group[K]{live: map[K]*Broadcast{}}
}

// Join returns the running broadcast for key, or starts one. started is true
// for the caller that has to produce the answer and call the returned done.
func (g *Group[K]) Join(key K) (b *Broadcast, started bool, done func()) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if b, ok := g.live[key]; ok {
		return b, false, func() {}
	}
	b = New()
	g.live[key] = b
	return b, true, func() {
		g.mu.Lock()
		delete(g.live, key)
		g.mu.Unlock()
	}
}
