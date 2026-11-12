package thread_summary

import (
	"context"
	"crypto/sha256"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/resources/post/thread_querier"
	thread_service "github.com/Southclaws/storyden/app/services/thread"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	MinReplies   = 5
	maxReplies   = 200
	maxCached    = 500
	modelTimeout = 90 * time.Second
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

	mu    sync.Mutex
	cache map[[32]byte]Summary
}

func Build() fx.Option {
	return fx.Provide(New)
}

func New(logger *slog.Logger, prompter ai.Prompter, threads thread_service.Service, assets *asset_querier.Querier) *Summariser {
	return &Summariser{logger: logger, prompter: prompter, threads: threads, assets: assets, cache: map[[32]byte]Summary{}}
}

func (s *Summariser) Summarise(ctx context.Context, threadID post.ID) (*Summary, error) {
	thr, err := s.threads.Get(ctx, threadID, pagination.NewPageParams(1, maxReplies), thread_querier.WithRepliesDescending(false))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if thr.Access.Ok() {
		return &Summary{}, nil
	}

	if len(thr.Replies.Items) < MinReplies {
		return &Summary{}, nil
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
		return &Summary{}, nil
	}

	key := sha256.Sum256([]byte(in.text))
	s.mu.Lock()
	cached, ok := s.cache[key]
	s.mu.Unlock()
	if ok {
		return &cached, nil
	}

	mctx, cancel := context.WithTimeout(ctx, modelTimeout)
	defer cancel()

	res, err := s.prompter.Prompt(mctx, prompt+in.text)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if res == nil {
		return &Summary{}, nil
	}

	html, err := sanitise(res.Answer)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := Summary{HTML: html, IncludesAttachments: in.attachments > 0}
	if html != "" {
		s.store(key, out)
	}
	return &out, nil
}

func (s *Summariser) store(key [32]byte, v Summary) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.cache) >= maxCached {
		clear(s.cache)
	}
	s.cache[key] = v
}

func sanitise(answer string) (string, error) {
	answer = strings.TrimSpace(answer)
	answer = strings.TrimPrefix(answer, "```html")
	answer = strings.TrimPrefix(answer, "```")
	answer = strings.TrimSuffix(answer, "```")
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return "", nil
	}

	content, err := datagraph.NewRichText(answer)
	if err != nil {
		return "", err
	}
	return content.HTML(), nil
}

const prompt = `Summarise this discussion thread from a German dentistry students' forum for someone who has not read it.

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
