package ocr

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/message"
	"github.com/Southclaws/storyden/internal/config"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
	infra_ocr "github.com/Southclaws/storyden/internal/infrastructure/ocr"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

// Build wires up the OCR processor. The fx.Invoke forces the processor to
// be instantiated (and its background subscription/backfill started) on
// boot regardless of what else happens to depend on it — without this, the
// processor previously only existed by accident of bindings.Admin injecting
// it.
func Build() fx.Option {
	return fx.Options(
		fx.Provide(NewProcessor),
		fx.Invoke(func(*Processor) {}),
	)
}

type Processor struct {
	logger       *slog.Logger
	cfg          config.Config
	ocrClient    infra_ocr.Client
	assetQuerier *asset_querier.Querier
	assetWriter  *asset_writer.Writer
	objects      object.Storer
	bus          *pubsub.Bus
	// sem bounds concurrent extraction work (tesseract/rasterisation are
	// CPU-bound external processes) across both the live bus handler and the
	// backfill loop. Sized by OCR_CONCURRENCY, which defaults to 1.
	sem chan struct{}
}

func NewProcessor(
	ctx context.Context,
	lc fx.Lifecycle,
	cfg config.Config,
	logger *slog.Logger,
	assetQuerier *asset_querier.Querier,
	assetWriter *asset_writer.Writer,
	objects object.Storer,
	bus *pubsub.Bus,
) *Processor {
	ocrClient := infra_ocr.New(cfg, logger)

	proc := &Processor{
		logger:       logger,
		cfg:          cfg,
		ocrClient:    ocrClient,
		assetQuerier: assetQuerier,
		assetWriter:  assetWriter,
		objects:      objects,
		bus:          bus,
		sem:          make(chan struct{}, concurrency(cfg)),
	}

	backfillCtx, cancelBackfill := context.WithCancel(ctx)

	lc.Append(fx.StartHook(func(hctx context.Context) error {
		if bus == nil {
			return nil
		}

		_, err := pubsub.SubscribeCommand(ctx, bus, "ocr.process_asset", func(ctx context.Context, cmd *message.CommandProcessAssetOCR) error {
			return proc.ProcessAsset(ctx, cmd.ID)
		})
		if err != nil {
			logger.Error("failed to subscribe to ocr.process_asset command", slog.String("error", err.Error()))
			return err
		}

		if cfg.OCREnabled && cfg.OCRBackfillEnabled {
			go proc.runBackfill(backfillCtx)
		}

		return nil
	}))

	lc.Append(fx.StopHook(func() error {
		cancelBackfill()
		return nil
	}))

	return proc
}

// runBackfill processes assets left in a pending/stuck state (typically from
// before this instance started, or from a crash) in bounded batches until
// none remain, then periodically re-checks in case new assets slip through.
func (p *Processor) runBackfill(ctx context.Context) {
	p.processPendingBatches(ctx)

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			p.processPendingBatches(ctx)
		}
	}
}

func (p *Processor) processPendingBatches(ctx context.Context) {
	for {
		if ctx.Err() != nil {
			return
		}

		n, err := p.processPendingBatch(ctx)
		if err != nil {
			p.logger.Error("OCR backfill batch failed", slog.String("error", err.Error()))
			return
		}
		if n == 0 {
			return
		}
	}
}

func (p *Processor) processPendingBatch(ctx context.Context) (int, error) {
	pending, err := p.assetQuerier.GetPendingOCR(ctx, p.cfg.OCRBackfillBatchSize, p.cfg.OCRStuckTimeout)
	if err != nil {
		return 0, fault.Wrap(err, fctx.With(ctx))
	}

	p.processConcurrently(ctx, pending)

	return len(pending), nil
}

// ProcessAllPending processes up to limit pending/stuck assets, returning
// how many were processed. Used by the admin reindex endpoint.
func (p *Processor) ProcessAllPending(ctx context.Context, limit int) (int, error) {
	pending, err := p.assetQuerier.GetPendingOCR(ctx, limit, p.cfg.OCRStuckTimeout)
	if err != nil {
		p.logger.Error("failed to query pending OCR assets for batch processing", slog.String("error", err.Error()))
		return 0, err
	}

	return p.processConcurrently(ctx, pending), nil
}

// processConcurrently runs extraction over ids using a worker pool sized by
// OCR_CONCURRENCY and returns how many succeeded. ProcessAsset still takes
// p.sem, so the pool never outruns the configured bound even when the live
// bus handler is working at the same time.
func (p *Processor) processConcurrently(ctx context.Context, ids []asset.AssetID) int {
	workers := concurrency(p.cfg)
	if workers > len(ids) {
		workers = len(ids)
	}
	if workers < 1 {
		return 0
	}

	queue := make(chan asset.AssetID)
	var processed atomic.Int64
	var wg sync.WaitGroup

	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for id := range queue {
				if err := p.ProcessAsset(ctx, xid.ID(id)); err != nil {
					p.logger.Warn("failed to process asset for text extraction", slog.String("id", id.String()), slog.String("error", err.Error()))
					continue
				}
				processed.Add(1)
			}
		}()
	}

	for _, id := range ids {
		if ctx.Err() != nil {
			break
		}
		queue <- id
	}
	close(queue)
	wg.Wait()

	return int(processed.Load())
}

// concurrency clamps OCR_CONCURRENCY to at least 1 so a zero or negative
// value in the environment cannot deadlock the pool.
func concurrency(cfg config.Config) int {
	if cfg.OCRConcurrency < 1 {
		return 1
	}
	return cfg.OCRConcurrency
}

func (p *Processor) ProcessAsset(ctx context.Context, id xid.ID) error {
	if !p.cfg.OCREnabled {
		p.logger.Debug("OCR processing skipped because OCR is disabled", slog.String("id", id.String()))
		return nil
	}

	select {
	case p.sem <- struct{}{}:
		defer func() { <-p.sem }()
	case <-ctx.Done():
		return ctx.Err()
	}

	a, err := p.assetQuerier.GetByID(ctx, id)
	if err != nil {
		p.logger.Error("failed to get asset for OCR processing", slog.String("id", id.String()), slog.String("error", err.Error()))
		return fault.Wrap(err, fctx.With(ctx))
	}

	mimeStr := a.MIME.String()
	if !p.isSupportedMIME(mimeStr) {
		p.logger.Debug("skipping unsupported asset type for OCR", slog.String("id", id.String()), slog.String("mime", mimeStr))
		_, err := p.assetWriter.UpdateOCRSkipped(ctx, id, fmt.Sprintf("unsupported mime type: %s", mimeStr))
		return err
	}

	maxBytes := int64(p.cfg.OCRMaxFileSizeMB) * 1024 * 1024
	if maxBytes > 0 && int64(a.Size) > maxBytes {
		p.logger.Warn("skipping OCR processing because file exceeds size limit", slog.String("id", id.String()), slog.Int("size", a.Size))
		_, err := p.assetWriter.UpdateOCRSkipped(ctx, id, fmt.Sprintf("file exceeds max size of %d MB", p.cfg.OCRMaxFileSizeMB))
		return err
	}

	path := asset.BuildAssetPath(a.Name)

	exists, err := p.objects.Exists(ctx, path)
	if err != nil {
		p.logger.Error("failed to check asset existence for OCR", slog.String("id", id.String()), slog.String("error", err.Error()))
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, fmt.Sprintf("existence check failed: %v", err))
		return fault.Wrap(err, fctx.With(ctx))
	}
	if !exists {
		p.logger.Warn("skipping OCR processing because asset file does not exist on disk", slog.String("id", id.String()), slog.String("path", path))
		_, err := p.assetWriter.UpdateOCRSkipped(ctx, id, "asset file not found on disk")
		return err
	}

	rc, _, err := p.objects.Read(ctx, path)
	if err != nil {
		p.logger.Error("failed to read asset object for OCR", slog.String("id", id.String()), slog.String("error", err.Error()))
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, fmt.Sprintf("read object failed: %v", err))
		return fault.Wrap(err, fctx.With(ctx))
	}
	if closer, ok := rc.(io.Closer); ok {
		defer closer.Close()
	}

	var reader io.Reader = rc
	if maxBytes > 0 {
		reader = io.LimitReader(rc, maxBytes+1)
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, fmt.Sprintf("read object failed: %v", err))
		return fault.Wrap(err, fctx.With(ctx))
	}

	p.logger.Info("starting text extraction for asset", slog.String("id", id.String()), slog.String("filename", a.Name.String()))
	_, _ = p.assetWriter.UpdateOCRProcessing(ctx, id)

	result, err := p.ocrClient.ExtractText(ctx, data, mimeStr)
	if err != nil {
		return p.handleExtractionError(ctx, id, err)
	}

	text := result.Text
	if p.cfg.OCRMaxTextLength > 0 && len(text) > p.cfg.OCRMaxTextLength {
		text = strings.ToValidUTF8(text[:p.cfg.OCRMaxTextLength], "")
	}

	p.logger.Info("text extraction completed successfully",
		slog.String("id", id.String()),
		slog.Int("text_length", len(text)),
		slog.String("engine", result.Engine),
	)

	_, err = p.assetWriter.UpdateOCRCompleted(ctx, id, text)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if p.bus != nil {
		p.bus.Publish(ctx, &message.EventAssetOCRCompleted{AssetID: id})
	}

	return nil
}

// handleExtractionError maps typed extraction errors onto the right terminal
// status: a missing engine binary or an unsupported/textless document is a
// permanent, expected condition (skipped, not retried), while anything else
// is a transient failure (failed, retried by the backfill).
func (p *Processor) handleExtractionError(ctx context.Context, id xid.ID, err error) error {
	switch {
	case errors.Is(err, infra_ocr.ErrEngineUnavailable):
		p.logger.Warn("OCR engine unavailable, skipping asset", slog.String("id", id.String()))
		_, werr := p.assetWriter.UpdateOCRSkipped(ctx, id, "ocr engine unavailable")
		return werr

	case errors.Is(err, infra_ocr.ErrUnsupportedMIME):
		_, werr := p.assetWriter.UpdateOCRSkipped(ctx, id, "unsupported mime type")
		return werr

	case errors.Is(err, infra_ocr.ErrNoTextLayer):
		_, werr := p.assetWriter.UpdateOCRSkipped(ctx, id, "pdf has no usable text layer and no rasteriser available")
		return werr

	default:
		p.logger.Error("text extraction failed for asset", slog.String("id", id.String()), slog.String("error", err.Error()))
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, err.Error())
		return fault.Wrap(err, fctx.With(ctx))
	}
}

// supportedOCRMIMETypes whitelists the exact formats Tesseract can handle.
// Asset filenames are slugified on upload (BuildAssetPath strips extensions
// entirely, e.g. "photo.png" -> "photo-png"), so filtering by filename
// extension isn't viable here — the MIME type, sniffed from file content on
// upload (see asset_upload.Uploader), is the only reliable signal. This is
// deliberately narrower than a generic "image/*" prefix match: that would
// still let image/svg+xml through, and Tesseract/Leptonica can't rasterise
// vector formats ("Leptonica Error in pixRead: image file not found").
var supportedOCRMIMETypes = map[string]bool{
	"image/png":       true,
	"image/jpeg":      true,
	"application/pdf": true,
}

// IsSupportedMIME reports whether an asset of this content type is worth
// enqueueing for text extraction at all. Upload sites use it to avoid
// dispatching commands that this package would immediately mark as skipped.
func IsSupportedMIME(m string) bool {
	return supportedOCRMIMETypes[strings.ToLower(strings.TrimSpace(m))]
}

func (p *Processor) isSupportedMIME(m string) bool {
	return IsSupportedMIME(m)
}
