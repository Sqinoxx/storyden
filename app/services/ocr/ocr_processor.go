package ocr

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/rs/xid"
	"go.uber.org/fx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/message"
	"github.com/Southclaws/storyden/internal/config"
	ent_asset "github.com/Southclaws/storyden/internal/ent/asset"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
	infra_ocr "github.com/Southclaws/storyden/internal/infrastructure/ocr"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
)

type Processor struct {
	logger       *slog.Logger
	cfg          config.Config
	ocrClient    infra_ocr.Client
	assetQuerier *asset_querier.Querier
	assetWriter  *asset_writer.Writer
	objects      object.Storer
	bus          *pubsub.Bus
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
	}

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

		return nil
	}))

	return proc
}

func (p *Processor) ProcessAllPending(ctx context.Context) (int, error) {
	pending, err := p.assetQuerier.GetPendingOCR(ctx, 1000)
	if err != nil {
		p.logger.Error("failed to query pending OCR assets for batch processing", slog.String("error", err.Error()))
		return 0, err
	}
	p.logger.Info("starting batch OCR processing for pending assets", slog.Int("count", len(pending)))
	processed := 0
	for _, a := range pending {
		err := p.ProcessAsset(ctx, xid.ID(a.ID))
		if err == nil {
			processed++
		}
	}
	p.logger.Info("batch OCR processing finished", slog.Int("processed", processed))
	return processed, nil
}

func (p *Processor) ProcessAsset(ctx context.Context, id xid.ID) error {
	if !p.cfg.OCREnabled {
		p.logger.Debug("OCR processing skipped because OCR is disabled", slog.String("id", id.String()))
		return nil
	}

	a, err := p.assetQuerier.GetByID(ctx, id)
	if err != nil {
		p.logger.Error("failed to get asset for OCR processing", slog.String("id", id.String()), slog.String("error", err.Error()))
		return fault.Wrap(err, fctx.With(ctx))
	}

	mimeStr := a.MIME.String()
	if !p.isSupportedMIME(mimeStr) {
		p.logger.Debug("skipping non-image asset for OCR", slog.String("id", id.String()), slog.String("mime", mimeStr))
		_, err := p.assetWriter.UpdateOCRStatus(ctx, id, ent_asset.OcrStatusSkipped)
		return err
	}

	// Check file size against max size limit
	maxBytes := int64(p.cfg.OCRMaxFileSizeMB) * 1024 * 1024
	if maxBytes > 0 && int64(a.Size) > maxBytes {
		p.logger.Warn("skipping OCR processing because file exceeds size limit", slog.String("id", id.String()), slog.Int("size", a.Size))
		_, err := p.assetWriter.UpdateOCRStatus(ctx, id, ent_asset.OcrStatusSkipped)
		return err
	}

	path := asset.BuildAssetPath(a.Name)
	rc, _, err := p.objects.Read(ctx, path)
	if err != nil {
		p.logger.Error("failed to read asset object for OCR", slog.String("id", id.String()), slog.String("error", err.Error()))
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, fmt.Sprintf("read object failed: %v", err))
		return fault.Wrap(err, fctx.With(ctx))
	}
	if closer, ok := rc.(io.Closer); ok {
		defer closer.Close()
	}

	p.logger.Info("starting OCR extraction for asset", slog.String("id", id.String()), slog.String("filename", a.Name.String()))
	_, _ = p.assetWriter.UpdateOCRStatus(ctx, id, ent_asset.OcrStatusProcessing)

	text, err := p.ocrClient.ExtractText(ctx, rc, mimeStr)
	if err != nil {
		p.logger.Error("OCR extraction failed for asset", slog.String("id", id.String()), slog.String("error", err.Error()))
		_, _ = p.assetWriter.UpdateOCRFailed(ctx, id, err.Error())
		return fault.Wrap(err, fctx.With(ctx))
	}

	p.logger.Info("OCR extraction completed successfully", slog.String("id", id.String()), slog.Int("text_length", len(text)))
	_, err = p.assetWriter.UpdateOCRCompleted(ctx, id, text)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if p.bus != nil {
		p.bus.Publish(ctx, &message.EventAssetOCRCompleted{AssetID: id})
	}

	return nil
}

func (p *Processor) isSupportedMIME(m string) bool {
	m = strings.ToLower(m)
	return strings.HasPrefix(m, "image/") || strings.Contains(m, "pdf")
}
