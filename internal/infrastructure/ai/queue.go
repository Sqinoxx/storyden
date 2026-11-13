package ai

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
)

const KindBusy ftag.Kind = "AI_BUSY"

var (
	ErrBusy      = fault.Wrap(fault.New("language model queue is full"), ftag.With(KindBusy), fmsg.WithDesc("busy", "Die KI ist gerade ausgelastet, bitte gleich nochmal versuchen."))
	ErrPreempted = errors.New("language model job preempted by a member request")
)

const (
	defaultIdleGap        = 30 * time.Second
	defaultMaxHighWaiting = 3
)

// Queue runs one language model job at a time. Jobs a member waits for run
// first and cancel a running background job, background jobs only start after
// no member job was seen for the idle gap.
type Queue struct {
	inner          Prompter
	idleGap        time.Duration
	maxHighWaiting int
	now            func() time.Time

	mu          sync.Mutex
	busy        bool
	runningLow  context.CancelCauseFunc
	highWaiting int
	lastHigh    time.Time
	wake        chan struct{}
}

func NewQueue(inner Prompter) *Queue {
	return &Queue{
		inner:          inner,
		idleGap:        defaultIdleGap,
		maxHighWaiting: defaultMaxHighWaiting,
		now:            time.Now,
		wake:           make(chan struct{}),
	}
}

func (q *Queue) broadcast() {
	close(q.wake)
	q.wake = make(chan struct{})
}

func (q *Queue) acquire(ctx context.Context, p Priority) (context.Context, func(), error) {
	q.mu.Lock()
	if p == PriorityHigh {
		if q.highWaiting >= q.maxHighWaiting {
			q.mu.Unlock()
			return nil, nil, fault.Wrap(ErrBusy, fctx.With(ctx))
		}
		q.highWaiting++
		q.lastHigh = q.now()
		if q.runningLow != nil {
			q.runningLow(ErrPreempted)
		}
	}

	for {
		var timer <-chan time.Time
		switch p {
		case PriorityHigh:
			if !q.busy {
				q.highWaiting--
				q.busy = true
				q.mu.Unlock()
				return ctx, q.releaser(PriorityHigh, nil), nil
			}
		case PriorityLow:
			idle := q.now().Sub(q.lastHigh)
			if !q.busy && q.highWaiting == 0 && idle >= q.idleGap {
				jctx, cancel := context.WithCancelCause(ctx)
				q.busy = true
				q.runningLow = cancel
				q.mu.Unlock()
				return jctx, q.releaser(PriorityLow, cancel), nil
			}
			if idle < q.idleGap {
				timer = time.After(q.idleGap - idle)
			}
		}

		wake := q.wake
		q.mu.Unlock()

		select {
		case <-ctx.Done():
			q.mu.Lock()
			if p == PriorityHigh {
				q.highWaiting--
			}
			q.mu.Unlock()
			return nil, nil, ctx.Err()
		case <-wake:
		case <-timer:
		}
		q.mu.Lock()
	}
}

func (q *Queue) releaser(p Priority, cancel context.CancelCauseFunc) func() {
	var once sync.Once
	return func() {
		once.Do(func() {
			q.mu.Lock()
			defer q.mu.Unlock()
			q.busy = false
			q.runningLow = nil
			if p == PriorityHigh {
				q.lastHigh = q.now()
			}
			if cancel != nil {
				cancel(nil)
			}
			q.broadcast()
		})
	}
}

func preempted(ctx context.Context, err error) error {
	if errors.Is(context.Cause(ctx), ErrPreempted) {
		return ErrPreempted
	}
	return err
}

func (q *Queue) Prompt(ctx context.Context, input string, opts ...Option) (*Result, error) {
	o := applyOptions(opts)
	jctx, release, err := q.acquire(ctx, o.Priority)
	if err != nil {
		return nil, err
	}
	defer release()

	res, err := q.inner.Prompt(jctx, input, opts...)
	if err != nil {
		return nil, preempted(jctx, err)
	}
	if o.Priority == PriorityLow && jctx.Err() != nil {
		return nil, preempted(jctx, jctx.Err())
	}
	return res, nil
}

func (q *Queue) PromptStream(ctx context.Context, input string, opts ...Option) (func(yield func(string, error) bool), error) {
	o := applyOptions(opts)
	jctx, release, err := q.acquire(ctx, o.Priority)
	if err != nil {
		return nil, err
	}

	stop := context.AfterFunc(jctx, release)

	inner, err := q.inner.PromptStream(jctx, input, opts...)
	if err != nil || inner == nil {
		stop()
		release()
		return inner, err
	}

	return func(yield func(string, error) bool) {
		defer release()
		defer stop()
		for chunk, err := range inner {
			if err != nil {
				err = preempted(jctx, err)
			}
			if !yield(chunk, err) {
				return
			}
		}
	}, nil
}

func (q *Queue) EmbeddingFunc() func(ctx context.Context, text string) ([]float32, error) {
	return q.inner.EmbeddingFunc()
}
