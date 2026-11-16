package thread_summary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"log/slog"
	"strings"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/thread_querier"
	"github.com/Southclaws/storyden/app/services/summary_stream"
	"github.com/Southclaws/storyden/app/services/summary_text"
	thread_service "github.com/Southclaws/storyden/app/services/thread"
	"github.com/Southclaws/storyden/internal/ent"
	ent_post "github.com/Southclaws/storyden/internal/ent/post"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	MinReplies    = 5
	maxReplies    = 200
	modelTimeout  = 90 * time.Second
	maxTokens     = 350
	temperature   = 0.2
	promptVersion = "thread-v1"
)

type Summary struct {
	HTML                string
	IncludesAttachments bool
}

type Summariser struct {
	logger   *slog.Logger
	prompter ai.Prompter
	threads  thread_service.Service
	assets   *asset_querier.Querier
	db       *ent.Client
	live     *summary_stream.Group[string]
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(logger *slog.Logger, prompter ai.Prompter, threads thread_service.Service, assets *asset_querier.Querier, db *ent.Client) *Summariser {
	return &Summariser{
		logger:   logger,
		prompter: prompter,
		threads:  threads,
		assets:   assets,
		db:       db,
		live:     summary_stream.NewGroup[string](),
	}
}

type job struct {
	id     xid.ID
	in     input
	hash   string
	stored string
}

func hashOf(text string) string {
	h := sha256.Sum256([]byte(promptVersion + "\x00" + text))
	return hex.EncodeToString(h[:16])
}

// prepare returns nil when the thread is hidden from the caller or too short.
func (s *Summariser) prepare(ctx context.Context, threadID post.ID) (*job, error) {
	thr, err := s.threads.Get(ctx, threadID, pagination.NewPageParams(1, maxReplies), thread_querier.WithRepliesDescending(false))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if thr.Access.Ok() {
		return nil, nil
	}
	if len(thr.Replies.Items) < MinReplies {
		return nil, nil
	}

	postIDs := []xid.ID{xid.ID(thr.ID)}
	for _, r := range thr.Replies.Items {
		postIDs = append(postIDs, xid.ID(r.ID))
	}
	attachments, err := s.assets.ListExtractedTextForPosts(ctx, postIDs)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	in := buildInput(thr, attachments)
	if in.replies < MinReplies {
		return nil, nil
	}

	j := &job{id: xid.ID(thr.ID), in: in, hash: hashOf(in.text)}

	row, err := s.db.Post.Query().
		Where(ent_post.ID(j.id)).
		Select(ent_post.FieldAiSummary, ent_post.FieldAiSummaryHash).
		Only(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if row.AiSummary != nil && row.AiSummaryHash != nil && *row.AiSummaryHash == j.hash {
		j.stored = *row.AiSummary
	}

	return j, nil
}

// Stream returns the stored summary at once or generates one live. Concurrent
// callers for the same thread content share one model call.
func (s *Summariser) Stream(ctx context.Context, threadID post.ID) (summary_stream.Stream, bool, error) {
	j, err := s.prepare(ctx, threadID)
	if err != nil {
		return nil, false, err
	}
	if j == nil {
		return summary_stream.Ready(""), false, nil
	}

	includes := j.in.attachments > 0
	if j.stored != "" {
		return summary_stream.Ready(j.stored), includes, nil
	}

	b, err := s.join(ctx, j)
	if err != nil {
		return nil, false, err
	}
	return b.Subscribe(ctx), includes, nil
}

func (s *Summariser) Summarise(ctx context.Context, threadID post.ID) (*Summary, error) {
	events, includes, err := s.Stream(ctx, threadID)
	if err != nil {
		return nil, err
	}

	for ev := range events {
		if !ev.Done {
			continue
		}
		if ev.Err != nil {
			return nil, fault.Wrap(ev.Err, fctx.With(ctx))
		}
		return &Summary{HTML: ev.HTML, IncludesAttachments: includes && ev.HTML != ""}, nil
	}

	return nil, fault.Wrap(ctx.Err(), fctx.With(ctx))
}

func (s *Summariser) join(ctx context.Context, j *job) (*summary_stream.Broadcast, error) {
	b, started, done := s.live.Join(j.id.String() + j.hash)
	if !started {
		return b, nil
	}

	gctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), modelTimeout)
	it, err := s.prompter.PromptStream(gctx, j.in.text, ai.WithSystem(instructions), ai.WithMaxTokens(maxTokens), ai.WithTemperature(temperature))
	if err != nil {
		cancel()
		b.Finish("", err)
		done()
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if it == nil {
		cancel()
		b.Finish("", nil)
		done()
		return b, nil
	}

	go func() {
		defer cancel()
		defer done()

		var answer strings.Builder
		for chunk, err := range it {
			if err != nil {
				s.logger.Warn("thread summary failed", slog.String("thread", j.id.String()), slog.String("error", err.Error()))
				b.Finish("", err)
				return
			}
			answer.WriteString(chunk)
			b.Push(chunk)
		}

		html, err := summary_text.SanitiseHTML(answer.String())
		if err != nil {
			b.Finish("", err)
			return
		}
		if html != "" {
			err := s.db.Post.UpdateOneID(j.id).SetAiSummary(html).SetAiSummaryHash(j.hash).Exec(gctx)
			if err != nil {
				s.logger.Error("failed to store thread summary", slog.String("thread", j.id.String()), slog.String("error", err.Error()))
			}
		}
		b.Finish(html, nil)
	}()

	return b, nil
}

const instructions = `Summarise this discussion thread from a German dentistry students' forum for someone who has not read it.

Rules:
- Write in the language of the thread.
- Output only simple HTML using <p>, <strong>, <ul> and <li>. No markdown, no code fences.
- Use these sections, each as <p><strong>Heading</strong></p> followed by a <ul> with 1 to 3 short bullet points. Translate the headings into the language of the thread:
  1. What it is about
  2. Key answers or solution
  3. From the attachments (only if attachment text is given below)
  4. Still open (only if questions remain unanswered)
- Only use facts from the thread. Do not invent anything.
`
