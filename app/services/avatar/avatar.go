package avatar

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"path"
	"strings"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"

	"github.com/Southclaws/storyden/app/resources/account"
	"github.com/Southclaws/storyden/internal/infrastructure/imagesafe"
	"github.com/Southclaws/storyden/internal/mime"
)

// maxAvatarUploadBytes bounds the declared and actual upload size, well above
// what any real profile photo needs but small enough that a bogus 2GB
// Content-Length claim, or a client that just keeps streaming past what it
// declared, can't be used to exhaust storage or memory.
const maxAvatarUploadBytes = 2 << 20 // 2MB

var (
	errBadFormat      = fault.Wrap(fault.New("bad format"), ftag.With(ftag.InvalidArgument))
	errAvatarTooLarge = fault.Wrap(fault.New("avatar exceeds the maximum upload size"), ftag.With(ftag.InvalidArgument))
)

func avatarPath(aid account.AccountID) string {
	return path.Join("avatar", aid.String())
}

func (s *service) Exists(ctx context.Context, accountID account.AccountID) bool {
	exists, err := s.storage.Exists(ctx, avatarPath(accountID))
	if err != nil {
		return false // errors are ignored for now 🤠
	}

	return exists
}

// Set previously wrote whatever bytes the client sent straight to storage and
// served them back with a hardcoded image/png content type - so an arbitrary
// file (e.g. HTML, or something crafted to exploit an image parser elsewhere)
// could be uploaded as an "avatar" and later fetched with a mismatched or
// misleading content type. Every avatar is now decoded and re-encoded as PNG,
// the same pattern the site icon upload already uses (see
// app/services/branding/icon/service.go), which also rejects anything that
// isn't a real, safely-sized image.
func (s *service) Set(ctx context.Context, accountID account.AccountID, stream io.Reader, size int64) error {
	if size > maxAvatarUploadBytes {
		return fault.Wrap(errAvatarTooLarge, fctx.With(ctx))
	}

	limited := io.LimitReader(stream, maxAvatarUploadBytes+1)

	mt, r, err := mime.Detect(limited)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if !strings.HasPrefix(mt.String(), "image") {
		return fault.Wrap(errBadFormat, fctx.With(ctx))
	}

	img, _, err := imagesafe.Decode(r)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	if err := s.storage.Write(ctx, avatarPath(accountID), &buf, int64(buf.Len())); err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	return nil
}

func (s *service) Get(ctx context.Context, accountID account.AccountID) (io.Reader, int64, error) {
	stream, size, err := s.storage.Read(ctx, avatarPath(accountID))
	if err != nil {
		r, w := io.Pipe()

		go func() {
			i, err := s.generator.Generate(ctx, accountID.String())
			if err != nil {
				w.CloseWithError(err)
				return
			}

			if err := png.Encode(w, i); err != nil {
				w.CloseWithError(err)
				return
			}

			w.Close()
		}()

		return r, 0, nil
	}

	return stream, size, nil
}
