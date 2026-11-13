package pdf_summary

import (
	"context"
	"sync"
)

type broadcast struct {
	mu     sync.Mutex
	chunks []string
	done   bool
	html   string
	err    error
	wake   chan struct{}
}

func newBroadcast() *broadcast {
	return &broadcast{wake: make(chan struct{})}
}

func (b *broadcast) notify() {
	close(b.wake)
	b.wake = make(chan struct{})
}

func (b *broadcast) push(chunk string) {
	if chunk == "" {
		return
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chunks = append(b.chunks, chunk)
	b.notify()
}

func (b *broadcast) finish(html string, err error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.done = true
	b.html = html
	b.err = err
	b.notify()
}

func (b *broadcast) subscribe(ctx context.Context) func(yield func(Event) bool) {
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
