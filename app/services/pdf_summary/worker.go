package pdf_summary

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/message"
	"github.com/Southclaws/storyden/app/resources/settings"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/infrastructure/ai"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

const (
	idlePoll   = time.Minute
	busyPoll   = 15 * time.Second
	jobTimeout = 5 * time.Minute
	nightStart = 1
	nightEnd   = 7
)

type Worker struct {
	svc      *Service
	settings *settings.SettingsRepository
	queued   bool
	wake     chan struct{}
}

func newWorker(ctx context.Context, lc fx.Lifecycle, svc *Service, settingsRepo *settings.SettingsRepository, bus *pubsub.Bus) *Worker {
	w := &Worker{svc: svc, settings: settingsRepo, wake: make(chan struct{}, 1)}
	if ai.IsDisabled(svc.prompter) {
		return w
	}

	wctx, cancel := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func(hctx context.Context) error {
		if bus != nil {
			_, err := pubsub.Subscribe(ctx, bus, "pdf_summary.asset_ocr_completed", func(ctx context.Context, evt *message.EventAssetOCRCompleted) error {
				return w.onOCRCompleted(ctx, evt.AssetID)
			})
			if err != nil {
				return err
			}
		}
		go w.run(wctx)
		return nil
	}))

	lc.Append(fx.StopHook(func() error {
		cancel()
		return nil
	}))

	return w
}

func (w *Worker) onOCRCompleted(ctx context.Context, id xid.ID) error {
	a, err := w.svc.assetQ.GetByID(ctx, asset.AssetID(id))
	if err != nil {
		return nil
	}
	if !isPDF(a) {
		return nil
	}
	if err := w.svc.assetW.SetSummaryStatus(ctx, id, ent_asset.AiSummaryStatusQueued); err != nil {
		return err
	}
	select {
	case w.wake <- struct{}{}:
	default:
	}
	return nil
}

func inNight(t time.Time) bool {
	h := t.Hour()
	return h >= nightStart && h < nightEnd
}

func (w *Worker) run(ctx context.Context) {
	for {
		wait := w.step(ctx)
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		case <-w.wake:
		}
	}
}

func (w *Worker) step(ctx context.Context) time.Duration {
	set, err := w.settings.Get(ctx)
	if err != nil {
		return idlePoll
	}
	if !set.AIPDFSummaryBackgroundEnabled() {
		return idlePoll
	}
	if set.AIPDFSummaryNightOnly() && !inNight(w.svc.now()) {
		return idlePoll
	}

	if !w.queued {
		n, err := w.svc.assetW.QueueUnsummarisedPDFs(ctx)
		if err != nil {
			return idlePoll
		}
		w.queued = true
		if n > 0 {
			w.svc.logger.Info("queued existing pdfs for summaries", slog.Int("count", n))
		}
	}

	if w.svc.cfg.OCREnabled {
		processing, err := w.svc.assetQ.CountProcessingOCR(ctx)
		if err == nil && processing > 0 {
			return busyPoll
		}
	}

	a, err := w.svc.assetQ.NextQueuedSummary(ctx)
	if err != nil || a == nil {
		return idlePoll
	}

	jctx, cancel := context.WithTimeout(ctx, jobTimeout)
	defer cancel()

	start := w.svc.now()
	err = w.svc.summariseFull(jctx, a)
	switch {
	case err == nil:
		w.svc.logger.Info("pdf summary prepared", slog.String("asset", a.ID.String()), slog.Duration("took", w.svc.now().Sub(start)))
		return 0
	case errors.Is(err, ai.ErrPreempted), errors.Is(err, context.Canceled):
		return busyPoll
	default:
		w.svc.logger.Warn("pdf summary failed", slog.String("asset", a.ID.String()), slog.String("error", err.Error()))
		_ = w.svc.assetW.SetSummaryStatus(ctx, xid.ID(a.ID), ent_asset.AiSummaryStatusFailed)
		return busyPoll
	}
}
