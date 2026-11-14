// Package tag_backfill runs the tag suggester over every published thread in
// the background and collects the tags it would add, for an admin to review.
// Tags are only ever added, never removed, and only tags that already exist.
package tag_backfill

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/post/thread_cache"
	"github.com/Southclaws/storyden/app/services/tag/tag_suggest"
	"github.com/Southclaws/storyden/internal/ent"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	ent_tag "github.com/Southclaws/storyden/internal/ent/tag"
)

const (
	pageSize       = 50
	maxAssetChars  = 1500
	maxAssetsRead  = 3
	threadTimeout  = 90 * time.Second
	maxApplyTagLen = 64
)

var ErrDisabled = fault.New("language model not configured", ftag.With(ftag.InvalidArgument))

type Proposal struct {
	ThreadID xid.ID
	Slug     string
	Title    string
	Current  []string
	Add      []string
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
	Tags     []string
}

type Backfill struct {
	logger    *slog.Logger
	db        *ent.Client
	suggester *tag_suggest.Suggester
	cache     *thread_cache.Cache
	root      context.Context

	mu     sync.Mutex
	status Status
	cancel context.CancelFunc
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(lc fx.Lifecycle, logger *slog.Logger, db *ent.Client, suggester *tag_suggest.Suggester, cache *thread_cache.Cache) *Backfill {
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
	if !b.suggester.Enabled() {
		return Status{}, fault.Wrap(ErrDisabled, fctx.With(ctx))
	}

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
			WithTags().
			WithAssets().
			Order(ent.Desc(ent_post.FieldCreatedAt), ent.Asc(ent_post.FieldID)).
			Offset(offset).
			Limit(pageSize).
			All(ctx)
		if err != nil {
			if ctx.Err() == nil {
				b.logger.Error("tag backfill: listing threads failed", slog.String("error", err.Error()))
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
				b.logger.Warn("tag backfill: thread failed", slog.String("thread", p.ID.String()), slog.String("error", err.Error()))
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

	suggested, err := b.suggester.SuggestExcluding(ctx, p.Title, threadText(p), p.ID)
	if err != nil {
		return nil, err
	}

	current := make([]string, 0, len(p.Edges.Tags))
	has := map[string]bool{}
	for _, t := range p.Edges.Tags {
		current = append(current, t.Name)
		has[strings.ToLower(t.Name)] = true
	}

	var add []string
	for _, t := range suggested {
		if !has[strings.ToLower(t)] {
			add = append(add, t)
		}
	}
	if len(add) == 0 {
		return nil, nil
	}

	return &Proposal{ThreadID: p.ID, Slug: p.Slug, Title: p.Title, Current: current, Add: add}, nil
}

func threadText(p *ent.Post) string {
	var parts []string

	if content, err := datagraph.NewRichTextWithBlocks(p.Body); err == nil {
		if plain := strings.TrimSpace(content.Plaintext()); plain != "" {
			parts = append(parts, plain)
		}
	}

	read := 0
	for _, a := range p.Edges.Assets {
		if read == maxAssetsRead {
			break
		}
		text := ""
		if a.OcrText != nil {
			text = strings.Join(strings.Fields(*a.OcrText), " ")
		}
		if r := []rune(text); len(r) > maxAssetChars {
			text = string(r[:maxAssetChars])
		}
		if text == "" && !strings.HasSuffix(strings.ToLower(a.Filename), ".pdf") {
			continue
		}
		read++
		parts = append(parts, strings.TrimSpace("Datei "+a.Filename+": "+text))
	}

	return strings.Join(parts, "\n\n")
}

// Apply adds the given existing tags to the given threads. Unknown tags are
// ignored so a review can never create new tags. The thread's updated time is
// left alone so re-tagging does not reorder thread lists.
func (b *Backfill) Apply(ctx context.Context, items []Item) (int, error) {
	names := map[string]bool{}
	for _, it := range items {
		for _, t := range it.Tags {
			if t != "" && len(t) <= maxApplyTagLen {
				names[t] = true
			}
		}
	}
	if len(names) == 0 {
		return 0, nil
	}

	list := make([]string, 0, len(names))
	for n := range names {
		list = append(list, n)
	}
	tags, err := b.db.Tag.Query().Where(ent_tag.NameIn(list...)).All(ctx)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}
	byName := make(map[string]xid.ID, len(tags))
	for _, t := range tags {
		byName[t.Name] = t.ID
	}

	applied := 0
	done := map[xid.ID]map[string]bool{}
	for _, it := range items {
		p, err := b.threads().Where(ent_post.ID(it.ThreadID)).WithTags().Only(ctx)
		if ent.IsNotFound(err) {
			continue
		}
		if err != nil {
			return applied, fault.Wrap(err, fctx.With(ctx))
		}

		existing := map[string]bool{}
		for _, t := range p.Edges.Tags {
			existing[t.Name] = true
		}

		changed := false
		for _, name := range it.Tags {
			id, ok := byName[name]
			if !ok || existing[name] {
				continue
			}
			if err := b.db.Tag.UpdateOneID(id).AddPostIDs(p.ID).Exec(ctx); err != nil {
				return applied, fault.Wrap(err, fctx.With(ctx))
			}
			existing[name] = true
			changed = true
			applied++
			if done[p.ID] == nil {
				done[p.ID] = map[string]bool{}
			}
			done[p.ID][name] = true
		}

		if changed {
			if err := b.cache.Invalidate(ctx, p.ID); err != nil {
				return applied, fault.Wrap(err, fctx.With(ctx))
			}
		}
	}

	b.forget(done)

	return applied, nil
}

func (b *Backfill) forget(done map[xid.ID]map[string]bool) {
	if len(done) == 0 {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	kept := b.status.Proposals[:0]
	for _, p := range b.status.Proposals {
		if added, ok := done[p.ThreadID]; ok {
			var rest []string
			for _, t := range p.Add {
				if !added[t] {
					rest = append(rest, t)
				} else {
					p.Current = append(append([]string{}, p.Current...), t)
				}
			}
			if len(rest) == 0 {
				continue
			}
			p.Add = rest
		}
		kept = append(kept, p)
	}
	b.status.Proposals = kept
}
