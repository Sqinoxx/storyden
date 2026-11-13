package pdf_summary

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/services/summary_text"
	thread_service "github.com/Southclaws/storyden/app/services/thread"
	"github.com/Southclaws/storyden/internal/config"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
)

const (
	promptVersion = "pdf-v1"
	minTextChars  = 300

	quickBudget    = 2500
	quickMaxTokens = 180
	liveTimeout    = 90 * time.Second

	fullBudget    = 6000
	fullMaxTokens = 300
)

type Status string

const (
	StatusReady       Status = "ready"
	StatusPending     Status = "pending"
	StatusUnavailable Status = "unavailable"
)

type Result struct {
	Status  Status
	Summary string
	Quality string
	Preview string
}

type Event struct {
	Text string
	HTML string
	Done bool
	Err  error
}

var (
	errNotFound  = fault.New("attachment not found in thread", ftag.With(ftag.NotFound))
	errNoText    = fault.New("no text could be extracted from this file", ftag.With(ftag.InvalidArgument))
	errEmptyLive = errors.New("language model returned an empty summary")
)

type Service struct {
	logger   *slog.Logger
	cfg      config.Config
	prompter ai.Prompter
	threads  thread_service.Service
	assetQ   *asset_querier.Querier
	assetW   *asset_writer.Writer
	now      func() time.Time

	mu   sync.Mutex
	live map[xid.ID]*broadcast
}

func Build() fx.Option {
	return fx.Options(
		fx.Provide(New, newWorker),
		fx.Invoke(func(*Worker) {}),
	)
}

func New(logger *slog.Logger, cfg config.Config, prompter ai.Prompter, threads thread_service.Service, assetQ *asset_querier.Querier, assetW *asset_writer.Writer) *Service {
	return &Service{
		logger:   logger,
		cfg:      cfg,
		prompter: prompter,
		threads:  threads,
		assetQ:   assetQ,
		assetW:   assetW,
		now:      time.Now,
		live:     map[xid.ID]*broadcast{},
	}
}

func isPDF(a *asset.Asset) bool {
	name := strings.ToLower(a.Name.String())
	return a.MIME.String() == "application/pdf" || strings.HasSuffix(name, ".pdf") || strings.HasSuffix(name, "-pdf")
}

func textOf(a *asset.Asset) (string, bool) {
	if a.OCRStatus != string(ent_asset.OcrStatusCompleted) {
		return "", false
	}
	t := strings.TrimSpace(a.OCRText.OrZero())
	return t, len([]rune(t)) >= minTextChars
}

func hashOf(text string) string {
	h := sha256.Sum256([]byte(promptVersion + "\x00" + text))
	return hex.EncodeToString(h[:16])
}

func stored(a *asset.Asset, text string) (asset.Summary, bool) {
	s, ok := a.Summary.Get()
	if !ok || s.Hash != hashOf(text) {
		return asset.Summary{}, false
	}
	return s, true
}

func (s *Service) load(ctx context.Context, threadID post.ID, assetID asset.AssetID) (*asset.Asset, error) {
	thr, err := s.threads.Get(ctx, threadID, pagination.NewPageParams(1, 1))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if thr.Access.Ok() {
		return nil, fault.Wrap(errNotFound, fctx.With(ctx))
	}

	a, err := s.assetQ.GetInThread(ctx, xid.ID(assetID), xid.ID(threadID))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.NotFound))
	}
	if !isPDF(a) {
		return nil, fault.Wrap(errNotFound, fctx.With(ctx))
	}
	return a, nil
}

func (s *Service) Get(ctx context.Context, threadID post.ID, assetID asset.AssetID) (*Result, error) {
	a, err := s.load(ctx, threadID, assetID)
	if err != nil {
		return nil, err
	}

	text, ok := textOf(a)
	if !ok {
		return &Result{Status: StatusUnavailable}, nil
	}

	sel := selectText(text, quickBudget)
	res := &Result{
		Status:  StatusPending,
		Preview: buildPreview(summary_text.DisplayName(a.Name.String()), text, sel, s.now()),
	}
	if sum, ok := stored(a, text); ok {
		res.Status = StatusReady
		res.Summary = sum.HTML
		res.Quality = sum.Quality
	}
	return res, nil
}

// Stream returns the stored summary at once or generates a quick one live.
// Concurrent callers for the same attachment share one model call.
func (s *Service) Stream(ctx context.Context, threadID post.ID, assetID asset.AssetID) (func(yield func(Event) bool), error) {
	a, err := s.load(ctx, threadID, assetID)
	if err != nil {
		return nil, err
	}

	text, ok := textOf(a)
	if !ok {
		return nil, fault.Wrap(errNoText, fctx.With(ctx))
	}

	if sum, ok := stored(a, text); ok {
		return func(yield func(Event) bool) {
			yield(Event{Done: true, HTML: sum.HTML})
		}, nil
	}

	b, err := s.join(ctx, a, text)
	if err != nil {
		return nil, err
	}
	return b.subscribe(ctx), nil
}

func (s *Service) join(ctx context.Context, a *asset.Asset, text string) (*broadcast, error) {
	id := xid.ID(a.ID)

	s.mu.Lock()
	if b, ok := s.live[id]; ok {
		s.mu.Unlock()
		return b, nil
	}
	b := newBroadcast()
	s.live[id] = b
	s.mu.Unlock()

	done := func() {
		s.mu.Lock()
		delete(s.live, id)
		s.mu.Unlock()
	}

	gctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), liveTimeout)
	sel := selectText(text, quickBudget)
	it, err := s.prompter.PromptStream(gctx, quickPrompt(a, sel), ai.WithFastModel(), ai.WithMaxTokens(quickMaxTokens))
	if err == nil && it == nil {
		err = fault.Wrap(errNoText, fctx.With(ctx))
	}
	if err != nil {
		cancel()
		b.finish("", err)
		done()
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	go func() {
		defer cancel()
		defer done()

		var answer strings.Builder
		for chunk, err := range it {
			if err != nil {
				b.finish("", err)
				return
			}
			answer.WriteString(chunk)
			b.push(chunk)
		}

		html, err := summary_text.SanitiseHTML(answer.String())
		if err == nil && html == "" {
			err = errEmptyLive
		}
		if err != nil {
			s.logger.Warn("live pdf summary failed", slog.String("asset", a.ID.String()), slog.String("error", err.Error()))
			b.finish("", err)
			return
		}

		if err := s.assetW.SetSummary(gctx, id, html, hashOf(text), ent_asset.AiSummaryQualityQuick); err != nil {
			s.logger.Error("failed to store pdf summary", slog.String("asset", a.ID.String()), slog.String("error", err.Error()))
		}
		b.finish(html, nil)
	}()

	return b, nil
}

func (s *Service) summariseFull(ctx context.Context, a *asset.Asset) error {
	text, ok := textOf(a)
	if !ok || !isPDF(a) {
		return s.assetW.SetSummaryStatus(ctx, xid.ID(a.ID), ent_asset.AiSummaryStatusSkipped)
	}
	if sum, ok := stored(a, text); ok && sum.Quality == string(ent_asset.AiSummaryQualityFull) {
		return s.assetW.SetSummaryStatus(ctx, xid.ID(a.ID), ent_asset.AiSummaryStatusDone)
	}

	sel := selectText(text, fullBudget)
	res, err := s.prompter.Prompt(ctx, fullPrompt(a, sel), ai.WithLowPriority(), ai.WithMaxTokens(fullMaxTokens))
	if err != nil {
		return err
	}
	if res == nil {
		return s.assetW.SetSummaryStatus(ctx, xid.ID(a.ID), ent_asset.AiSummaryStatusSkipped)
	}

	html, err := summary_text.SanitiseHTML(res.Answer)
	if err != nil || html == "" {
		return s.assetW.SetSummaryStatus(ctx, xid.ID(a.ID), ent_asset.AiSummaryStatusFailed)
	}
	return s.assetW.SetSummary(ctx, xid.ID(a.ID), html, hashOf(text), ent_asset.AiSummaryQualityFull)
}
