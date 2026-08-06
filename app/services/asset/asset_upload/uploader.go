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

	a, err := func() (asset *asset.Asset, err error) {
		if pid, ok := opts.ParentID.Get(); ok {
			return s.assets.AddVersion(ctx, xid.ID(accountID), name, int(size), *mt, pid)
		} else {
			return s.assets.Add(ctx, xid.ID(accountID), name, int(size), *mt)
		}
	}()
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	path := asset.BuildAssetPath(a.Name)

	if err := s.objects.Write(ctx, path, r, size); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if s.bus != nil {
		if err := s.bus.SendCommand(ctx, &message.CommandProcessAssetOCR{ID: xid.ID(a.ID)}); err != nil {
			s.logger.Warn("failed to dispatch OCR processing command for asset", slog.String("id", a.ID.String()), slog.String("error", err.Error()))
		}
	}

	return a, nil
}
