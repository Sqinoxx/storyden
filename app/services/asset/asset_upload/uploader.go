package asset_upload

import (
	"context"
	"io"
	"log/slog"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_writer"
	"github.com/Southclaws/storyden/app/resources/library/node_writer"
	"github.com/Southclaws/storyden/app/resources/message"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/ocr"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
	"github.com/Southclaws/storyden/internal/infrastructure/pubsub"
	"github.com/Southclaws/storyden/internal/mime"
)

type Uploader struct {
	logger     *slog.Logger
	nodewriter *node_writer.Writer
	assets     *asset_writer.Writer
	objects    object.Storer
	bus        *pubsub.Bus
}

func New(
	logger *slog.Logger,

	nodewriter *node_writer.Writer,
	assets *asset_writer.Writer,
	objects object.Storer,
	bus *pubsub.Bus,
) *Uploader {
	return &Uploader{
		logger:     logger,
		nodewriter: nodewriter,
		assets:     assets,
		objects:    objects,
		bus:        bus,
	}
}

type Options struct {
	ParentID opt.Optional[asset.AssetID]
}

func (s *Uploader) Upload(ctx context.Context, or io.Reader, size int64, name asset.Filename, opts Options) (*asset.Asset, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	mt, r, err := mime.Detect(or)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	// The declared size comes from a client-supplied Content-Length header, so
	// it is a bound rather than a fact. Cap the stream at it and record what was
	// actually stored, otherwise the DB size and the object disagree whenever a
	// client lies or the connection drops mid-body.
	counter := &countingReader{r: io.LimitReader(r, size)}

	// The object is written before the row exists so that a failed write leaves
	// nothing behind. A row without an object is permanently broken (AssetGet
	// 404s from storage, OCR can never process it); an object without a row is
	// merely unreferenced and gets cleaned up below.
	path := asset.BuildAssetPath(name)

	if err := s.objects.Write(ctx, path, counter, size); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	written := counter.n

	a, err := func() (asset *asset.Asset, err error) {
		if pid, ok := opts.ParentID.Get(); ok {
			return s.assets.AddVersion(ctx, xid.ID(accountID), name, int(written), *mt, pid)
		} else {
			return s.assets.Add(ctx, xid.ID(accountID), name, int(written), *mt)
		}
	}()
	if err != nil {
		if derr := s.objects.Delete(ctx, path); derr != nil {
			s.logger.Warn("failed to clean up object after asset record creation failed",
				slog.String("path", path),
				slog.String("error", derr.Error()),
			)
		}
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if s.bus != nil && ocr.IsSupportedMIME(mt.String()) {
		if err := s.bus.SendCommand(ctx, &message.CommandProcessAssetOCR{ID: xid.ID(a.ID)}); err != nil {
			s.logger.Warn("failed to dispatch OCR processing command for asset", slog.String("id", a.ID.String()), slog.String("error", err.Error()))
		}
	}

	return a, nil
}

type countingReader struct {
	r io.Reader
	n int64
}

func (c *countingReader) Read(p []byte) (int, error) {
	n, err := c.r.Read(p)
	c.n += int64(n)
	return n, err
}
