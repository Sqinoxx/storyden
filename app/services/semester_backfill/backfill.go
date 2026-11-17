// Package semester_backfill runs the semester suggester over every published
// thread in the background, relative to when each thread was posted, and
// collects the threads whose semester differs from the detected one for an
// admin to review.
package semester_backfill

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post/thread_cache"
	"github.com/Southclaws/storyden/app/services/account/semester"
	"github.com/Southclaws/storyden/app/services/semester_suggest"
	"github.com/Southclaws/storyden/internal/ent"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
)

const (
	pageSize      = 50
	threadTimeout = 60 * time.Second
	metaKey       = "semester"
)

type Proposal struct {
	ThreadID        xid.ID
	Slug            string
	Title           string
	CreatedAt       time.Time
	Current         semester.Term
	CurrentExplicit bool
	Suggested       semester.Term
	Source          semester_suggest.Source
	Evidence        string
}

type Status struct {
	Running    bool
	Checked    int
	Total      int
	Failed     int
	StartedAt  time.Time
	FinishedAt time.Time
	Error      string
	Proposals  []Proposal
}

type Item struct {
	ThreadID xid.ID
	Term     semester.Term
}

type Backfill struct {
	logger    *slog.Logger
	db        *ent.Client
	suggester *semester_suggest.Suggester
	cache     *thread_cache.Cache
	root      context.Context

	mu     sync.Mutex
	status Status
	cancel context.CancelFunc
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(lc fx.Lifecycle, logger *slog.Logger, db *ent.Client, suggester *semester_suggest.Suggester, cache *thread_cache.Cache) *Backfill {
	root, stop := context.WithCancel(context.Background())
	lc.Append(fx.StopHook(stop))

	return &Backfill{
		logger:    logger,
		db:        db,
		suggester: suggester,
		cache:     cache,
		root:      root,
	}
}

func (b *Backfill) Status() Status {
	b.mu.Lock()
	defer b.mu.Unlock()

	s := b.status
	s.Proposals = append([]Proposal{}, b.status.Proposals...)
	return s
}

func (b *Backfill) Start(ctx context.Context) (Status, error) {
	total, err := b.threads().Count(ctx)
	if err != nil {
		return Status{}, fault.Wrap(err, fctx.With(ctx))
	}

	b.mu.Lock()
	if b.status.Running {
		b.mu.Unlock()
		return b.Status(), nil
	}
	run, cancel := context.WithCancel(b.root)
	b.cancel = cancel
	b.status = Status{Running: true, Total: total, StartedAt: time.Now(), Proposals: []Proposal{}}
	b.mu.Unlock()

	go b.run(run)

	return b.Status(), nil
}

func (b *Backfill) Cancel() Status {
	b.mu.Lock()
	if b.cancel != nil {
		b.cancel()
	}
	b.mu.Unlock()
	return b.Status()
}

func (b *Backfill) threads() *ent.PostQuery {
	return b.db.Post.Query().Where(
		ent_post.RootPostIDIsNil(),
		ent_post.DeletedAtIsNil(),
		ent_post.VisibilityEQ(ent_post.VisibilityPublished),
	)
}

func (b *Backfill) run(ctx context.Context) {
	defer func() {
		b.mu.Lock()
		b.status.Running = false
		b.status.FinishedAt = time.Now()
		if ctx.Err() != nil && b.status.Error == "" {
			b.status.Error = "cancelled"
		}
		b.cancel = nil
		b.mu.Unlock()
	}()

	for offset := 0; ; offset += pageSize {
		page, err := b.threads().
			WithAssets().
			Order(ent.Desc(ent_post.FieldCreatedAt), ent.Asc(ent_post.FieldID)).
			Offset(offset).
			Limit(pageSize).
			All(ctx)
		if err != nil {
			if ctx.Err() == nil {
				b.logger.Error("semester backfill: listing threads failed", slog.String("error", err.Error()))
				b.mu.Lock()
				b.status.Error = err.Error()
				b.mu.Unlock()
			}
			return
		}

		for _, p := range page {
			if ctx.Err() != nil {
				return
			}
			proposal, err := b.propose(ctx, p)

			b.mu.Lock()
			b.status.Checked++
			if err != nil {
				b.status.Failed++
			} else if proposal != nil {
				b.status.Proposals = append(b.status.Proposals, *proposal)
			}
			b.mu.Unlock()

			if err != nil && ctx.Err() == nil {
				b.logger.Warn("semester backfill: thread failed", slog.String("thread", p.ID.String()), slog.String("error", err.Error()))
			}
		}

		if len(page) < pageSize {
			return
		}
	}
}

func (b *Backfill) propose(ctx context.Context, p *ent.Post) (*Proposal, error) {
	ctx, cancel := context.WithTimeout(ctx, threadTimeout)
	defer cancel()

	sug, err := b.suggester.SuggestAt(ctx, threadInput(p), p.CreatedAt)
	if err != nil {
		return nil, err
	}
	if sug == nil {
		return nil, nil
	}

	current, explicit := ThreadTerm(p.Metadata, p.CreatedAt)
	if current == sug.Term {
		return nil, nil
	}

	return &Proposal{
		ThreadID:        p.ID,
		Slug:            p.Slug,
		Title:           p.Title,
		CreatedAt:       p.CreatedAt,
		Current:         current,
		CurrentExplicit: explicit,
		Suggested:       sug.Term,
		Source:          sug.Source,
		Evidence:        sug.Evidence,
	}, nil
}

func threadInput(p *ent.Post) semester_suggest.Input {
	in := semester_suggest.Input{Title: p.Title}

	if content, err := datagraph.NewRichTextWithBlocks(p.Body); err == nil {
		in.Body = strings.TrimSpace(content.Plaintext())
	}

	for _, a := range p.Edges.Assets {
		in.Filenames = append(in.Filenames, a.Filename)
	}

	return in
}

// ThreadTerm mirrors the frontend: the explicitly chosen semester if there is
// one, otherwise the term the thread was posted in.
func ThreadTerm(meta map[string]any, createdAt time.Time) (semester.Term, bool) {
	if raw, ok := meta[metaKey].(map[string]any); ok {
		if key, ok := raw["term"].(string); ok {
			if t, err := semester.Parse(key); err == nil {
				return t, true
			}
		}
	}

	return semester.TermFor(createdAt), false
}

// Apply writes the given semesters into the threads' metadata. The thread's
// updated time is left alone so corrections do not reorder thread lists.
func (b *Backfill) Apply(ctx context.Context, items []Item) (int, error) {
	applied := 0
	done := map[xid.ID]semester.Term{}

	for _, it := range items {
		p, err := b.threads().Where(ent_post.ID(it.ThreadID)).Only(ctx)
		if ent.IsNotFound(err) {
			continue
		}
		if err != nil {
			return applied, fault.Wrap(err, fctx.With(ctx))
		}

		if current, explicit := ThreadTerm(p.Metadata, p.CreatedAt); explicit && current == it.Term {
			done[p.ID] = it.Term
			continue
		}

		meta := make(map[string]any, len(p.Metadata)+1)
		for k, v := range p.Metadata {
			meta[k] = v
		}
		meta[metaKey] = map[string]any{"term": it.Term.String()}

		if err := b.db.Post.UpdateOneID(p.ID).
			SetMetadata(meta).
			SetUpdatedAt(p.UpdatedAt).
			Exec(ctx); err != nil {
			return applied, fault.Wrap(err, fctx.With(ctx))
		}

		if err := b.cache.Invalidate(ctx, p.ID); err != nil {
			return applied, fault.Wrap(err, fctx.With(ctx))
		}

		applied++
		done[p.ID] = it.Term
	}

	b.forget(done)

	return applied, nil
}

func (b *Backfill) forget(done map[xid.ID]semester.Term) {
	if len(done) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	kept := b.status.Proposals[:0]
	for _, p := range b.status.Proposals {
		if _, ok := done[p.ThreadID]; ok {
			continue
		}
		kept = append(kept, p)
	}
	b.status.Proposals = kept
}
