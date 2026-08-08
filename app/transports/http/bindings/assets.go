package bindings

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/cachecontrol"
	"github.com/Southclaws/storyden/app/services/asset/asset_download"
	"github.com/Southclaws/storyden/app/services/asset/asset_upload"
	"github.com/Southclaws/storyden/app/services/reqinfo"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type Assets struct {
	uploader   *asset_upload.Uploader
	downloader *asset_download.Downloader
}

func NewAssets(uploader *asset_upload.Uploader, downloader *asset_download.Downloader) Assets {
	return Assets{uploader, downloader}
}

// assetCacheControl is long because asset bytes are immutable: the filename
// embeds the ID of the row that created it, so replacing content means a new
// URL. The validators below exist for clients whose entry has aged out and for
// intermediary caches that revalidate regardless.
const assetCacheControl = "public, max-age=31536000, immutable"

func (i *Assets) AssetGet(ctx context.Context, request openapi.AssetGetRequestObject) (openapi.AssetGetResponseObject, error) {
	filename := asset.NewFilepathFilename(request.AssetFilename)

	a, err := i.downloader.Head(ctx, filename)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	// Asset IDs are xids, which embed their creation timestamp. Since the bytes
	// behind an ID never change, that timestamp is a sound validator and saves
	// carrying separate columns through the download query.
	created := a.ID.Time()

	etag, notModified := reqinfo.GetCacheQuery(ctx).Check(func() *time.Time { return &created })
	if notModified {
		// Answered without touching storage at all.
		return openapi.AssetGet304Response{
			Headers: openapi.AssetNotModifiedResponseHeaders{
				CacheControl: assetCacheControl,
				ETag:         etag.String(),
				LastModified: cachecontrol.HTTPDate(etag.Time),
			},
		}, nil
	}

	r, err := i.downloader.Open(ctx, a)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	headers := assetDownloadHeaders(a, etag)

	rng, ok := reqinfo.GetRange(ctx).Get()
	if !ok {
		return openapi.AssetGet200AsteriskResponse{
			AssetDownloadOKAsteriskResponse: openapi.AssetDownloadOKAsteriskResponse{
				Body:          r,
				ContentType:   a.MIME.String(),
				ContentLength: int64(a.Size),
				Headers:       headers,
			},
		}, nil
	}

	total := int64(a.Size)

	start, end, err := parseByteRange(rng, total)
	if errors.Is(err, errUnsatisfiableRange) {
		closeReader(r)
		return openapi.AssetGet416Response{
			Headers: openapi.AssetRangeNotSatisfiableResponseHeaders{
				ContentRange: fmt.Sprintf("bytes */%d", total),
			},
		}, nil
	}

	seeker, seekable := r.(io.Seeker)

	// A range we cannot express, or a reader we cannot seek: RFC 7233 lets the
	// server ignore the header and send everything, which any client can use.
	// Refusing outright would leave it with nothing.
	if err != nil || !seekable {
		return openapi.AssetGet200AsteriskResponse{
			AssetDownloadOKAsteriskResponse: openapi.AssetDownloadOKAsteriskResponse{
				Body:          r,
				ContentType:   a.MIME.String(),
				ContentLength: total,
				Headers:       headers,
			},
		}, nil
	}

	if _, err := seeker.Seek(start, io.SeekStart); err != nil {
		closeReader(r)
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AssetGet206AsteriskResponse{
		AssetDownloadPartialAsteriskResponse: openapi.AssetDownloadPartialAsteriskResponse{
			Body:          newLimitedReadCloser(r, end-start+1),
			ContentType:   a.MIME.String(),
			ContentLength: end - start + 1,
			Headers: openapi.AssetDownloadPartialResponseHeaders{
				AcceptRanges:        headers.AcceptRanges,
				CacheControl:        headers.CacheControl,
				ContentDisposition:  headers.ContentDisposition,
				ContentRange:        fmt.Sprintf("bytes %d-%d/%d", start, end, total),
				ETag:                headers.ETag,
				LastModified:        headers.LastModified,
				XContentTypeOptions: headers.XContentTypeOptions,
			},
		},
	}, nil
}

func assetDownloadHeaders(a *asset.Asset, etag *cachecontrol.ETag) openapi.AssetDownloadOKResponseHeaders {
	return openapi.AssetDownloadOKResponseHeaders{
		AcceptRanges:       "bytes",
		CacheControl:       assetCacheControl,
		ContentDisposition: contentDisposition(a),
		ETag:               etag.String(),
		LastModified:       cachecontrol.HTTPDate(etag.Time),
		// The MIME type is sniffed from user-supplied content and this endpoint
		// is public, so the browser must not be allowed to reinterpret it.
		XContentTypeOptions: "nosniff",
	}
}

// inlineRenderableMIMEs are the types a browser may render in place. Everything
// else is forced to download rather than execute in the API origin, which
// matters because uploads are user-supplied and served unauthenticated.
var inlineRenderableMIMEs = map[string]bool{
	"application/pdf": true,
	"image/apng":      true,
	"image/avif":      true,
	"image/bmp":       true,
	"image/gif":       true,
	"image/jpeg":      true,
	"image/png":       true,
	"image/webp":      true,
	"video/mp4":       true,
	"video/webm":      true,
	"audio/mpeg":      true,
	"audio/ogg":       true,
	"audio/wav":       true,
}

func contentDisposition(a *asset.Asset) string {
	name := strings.ReplaceAll(a.Name.String(), `"`, "")

	if inlineRenderableMIMEs[strings.ToLower(a.MIME.String())] {
		return fmt.Sprintf(`inline; filename="%s"`, name)
	}

	return fmt.Sprintf(`attachment; filename="%s"`, name)
}

var (
	// errUnsatisfiableRange means the client asked for a range that cannot
	// exist in this resource, which must be answered with 416.
	errUnsatisfiableRange = fault.New("unsatisfiable range")

	// errUnsupportedRange means the request is well-formed but beyond what
	// this handler implements (multipart ranges). RFC 7233 permits ignoring
	// the header entirely in that case, so the caller sends the full body.
	errUnsupportedRange = fault.New("unsupported range")
)

// parseByteRange handles the single-range form of RFC 7233 that browsers and
// PDF viewers actually send, returning an inclusive start and end offset.
func parseByteRange(header string, size int64) (int64, int64, error) {
	spec, ok := strings.CutPrefix(strings.TrimSpace(header), "bytes=")
	if !ok {
		return 0, 0, errUnsupportedRange
	}

	if strings.Contains(spec, ",") {
		return 0, 0, errUnsupportedRange
	}

	first, last, ok := strings.Cut(spec, "-")
	if !ok {
		return 0, 0, errUnsupportedRange
	}

	first = strings.TrimSpace(first)
	last = strings.TrimSpace(last)

	if first == "" {
		// Suffix form: the last N bytes.
		n, err := strconv.ParseInt(last, 10, 64)
		if err != nil {
			return 0, 0, errUnsupportedRange
		}
		if n <= 0 {
			return 0, 0, errUnsatisfiableRange
		}
		if n > size {
			n = size
		}
		return size - n, size - 1, nil
	}

	start, err := strconv.ParseInt(first, 10, 64)
	if err != nil {
		return 0, 0, errUnsupportedRange
	}
	if start < 0 || start >= size {
		return 0, 0, errUnsatisfiableRange
	}

	end := size - 1
	if last != "" {
		end, err = strconv.ParseInt(last, 10, 64)
		if err != nil {
			return 0, 0, errUnsupportedRange
		}
		if end < start {
			return 0, 0, errUnsatisfiableRange
		}
		if end > size-1 {
			end = size - 1
		}
	}

	return start, end, nil
}

func closeReader(r io.Reader) {
	if c, ok := r.(io.Closer); ok {
		c.Close()
	}
}

// newLimitedReadCloser bounds a body to n bytes while preserving the underlying
// Close, which the generated response visitor relies on to release the object.
func newLimitedReadCloser(r io.Reader, n int64) io.Reader {
	c, ok := r.(io.Closer)
	if !ok {
		return io.LimitReader(r, n)
	}

	return struct {
		io.Reader
		io.Closer
	}{io.LimitReader(r, n), c}
}

func (i *Assets) AssetUpload(ctx context.Context, request openapi.AssetUploadRequestObject) (openapi.AssetUploadResponseObject, error) {
	// This route bypasses the OpenAPI validator so the body can be streamed,
	// which means its authorisation must be applied here instead.
	if err := AuthoriseOperation(ctx, "AssetUpload"); err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	name := opt.NewPtr(request.Params.Filename)

	// NOTE: Should we enforce a filename for upload? If none is available, the
	// client can decide on a suitable placeholder or generate a nonsense slug.
	filename := asset.NewFilename(name.Or("untitled"))

	parentID := opt.NewPtrMap(request.Params.ParentAssetId, deserialiseAssetID)

	opts := asset_upload.Options{
		ParentID: parentID,
	}

	a, err := i.uploader.Upload(ctx, request.Body, request.Params.ContentLength, filename, opts)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.AssetUpload200JSONResponse{
		AssetUploadOKJSONResponse: openapi.AssetUploadOKJSONResponse(serialiseAssetPtr(a)),
	}, nil
}

func serialiseAsset(a asset.Asset) openapi.Asset {
	path := fmt.Sprintf(`/api/assets/%s`, a.Name.String())

	return openapi.Asset{
		Id:       a.ID.String(),
		Filename: a.Name.String(),
		Path:     path,
		Parent:   opt.Map(a.Parent, serialiseAsset).Ptr(),
		MimeType: a.MIME.String(),
		Width:    float32(a.Metadata.GetWidth()),
		Height:   float32(a.Metadata.GetHeight()),
	}
}

func serialiseAssetPtr(a *asset.Asset) openapi.Asset {
	return serialiseAsset(*a)
}

func deserialiseAssetID(in string) asset.AssetID {
	return asset.AssetID(openapi.ParseID(in))
}

func deserialiseAssetIDs(ids []string) []asset.AssetID {
	return dt.Map(ids, deserialiseAssetID)
}
