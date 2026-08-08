package asset_download

import (
	"context"
	"io"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/asset/asset_querier"
	"github.com/Southclaws/storyden/internal/infrastructure/object"
)

type Downloader struct {
	assets  *asset_querier.Querier
	objects object.Storer
}

func New(
	assets *asset_querier.Querier,
	objects object.Storer,
) *Downloader {
	return &Downloader{
		assets:  assets,
		objects: objects,
	}
}

// Head resolves an asset's metadata without opening the underlying object. Use
// it to answer conditional requests, where the bytes are never sent and the
// storage round-trip would be wasted.
func (d *Downloader) Head(ctx context.Context, id asset.Filename) (*asset.Asset, error) {
	a, err := d.assets.GetForDownload(ctx, id)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return a, nil
}

// Open reads the bytes for an asset already resolved by Head, so a caller that
// needed the metadata first does not pay for a second lookup. It corrects Size
// to what storage actually holds, which is what Content-Length and range
// arithmetic must be based on.
func (d *Downloader) Open(ctx context.Context, a *asset.Asset) (io.Reader, error) {
	path := asset.BuildAssetPath(a.Name)
	ctx = fctx.WithMeta(ctx, "path", path, "asset_id", a.ID.String())

	r, size, err := d.objects.Read(ctx, path)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	a.Size = int(size)

	return r, nil
}

func (d *Downloader) Get(ctx context.Context, id asset.Filename) (*asset.Asset, io.Reader, error) {
	a, err := d.Head(ctx, id)
	if err != nil {
		return nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	r, err := d.Open(ctx, a)
	if err != nil {
		return nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	return a, r, nil
}

func (d *Downloader) GetByID(ctx context.Context, id asset.AssetID) (*asset.Asset, io.Reader, error) {
	a, err := d.assets.GetByID(ctx, id)
	if err != nil {
		return nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	path := asset.BuildAssetPath(a.Name)
	ctx = fctx.WithMeta(ctx, "path", path, "asset_id", id.String())

	r, size, err := d.objects.Read(ctx, path)
	if err != nil {
		return nil, nil, fault.Wrap(err, fctx.With(ctx))
	}

	a.Size = int(size)

	return a, r, nil
}
