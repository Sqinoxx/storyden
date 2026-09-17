package avatar

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"io"
	"testing"

	"github.com/Southclaws/fault/ftag"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/app/resources/account"
)

type stubGenerator struct{}

func (stubGenerator) Generate(ctx context.Context, seed string) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	return img, nil
}

type missingStorer struct{ errNotFound error }

func (m missingStorer) Exists(ctx context.Context, path string) (bool, error) { return false, nil }
func (m missingStorer) Read(ctx context.Context, path string) (io.Reader, int64, error) {
	return nil, 0, m.errNotFound
}
func (m missingStorer) Write(ctx context.Context, path string, r io.Reader, size int64) error {
	return nil
}
func (m missingStorer) Delete(ctx context.Context, path string) error { return nil }
func (m missingStorer) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}

// capturingStorer records whatever bytes Write is called with, so a test can
// assert on what actually got persisted rather than just that no error
// occurred.
type capturingStorer struct {
	missingStorer
	written []byte
}

func (c *capturingStorer) Write(ctx context.Context, path string, r io.Reader, size int64) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	c.written = data
	return nil
}

// TestGet_GeneratedAvatarIsComplete covers accounts with no uploaded avatar,
// where the image is produced on the fly and piped to the response. Closing the
// wrong end of that pipe made the consumer see io.ErrClosedPipe instead of EOF,
// truncating the PNG after the response headers had already been sent.
func TestGet_GeneratedAvatarIsComplete(t *testing.T) {
	r := require.New(t)

	s := &service{
		generator: stubGenerator{},
		storage:   missingStorer{errNotFound: io.ErrUnexpectedEOF},
	}

	reader, _, err := s.Get(context.Background(), account.AccountID{})
	r.NoError(err)

	data, err := io.ReadAll(reader)
	r.NoError(err, "the generated avatar must stream to completion")
	r.NotEmpty(data)

	img, err := png.Decode(bytes.NewReader(data))
	r.NoError(err, "the streamed bytes must be a complete, decodable PNG")
	r.Equal(8, img.Bounds().Dx())
	r.Equal(8, img.Bounds().Dy())
}

// TestSet_RejectsNonImageContent covers B18: Set used to write whatever bytes
// the client sent straight to storage with no validation at all, so an
// arbitrary file uploaded as an "avatar" would later be served back with a
// hardcoded image/png content type.
func TestSet_RejectsNonImageContent(t *testing.T) {
	r := require.New(t)

	storage := &capturingStorer{}
	s := &service{storage: storage}

	text := []byte("<html><body><script>alert(1)</script></body></html>")

	err := s.Set(context.Background(), account.AccountID{}, bytes.NewReader(text), int64(len(text)))
	r.Error(err)
	assert.Equal(t, ftag.InvalidArgument, ftag.Get(err))
	r.Nil(storage.written, "a rejected upload must not reach storage")
}

// TestSet_RejectsOversizedUpload covers the 2MB cap: a lying Content-Length
// must not be trusted to allow an unbounded write.
func TestSet_RejectsOversizedUpload(t *testing.T) {
	r := require.New(t)

	storage := &capturingStorer{}
	s := &service{storage: storage}

	err := s.Set(context.Background(), account.AccountID{}, bytes.NewReader(nil), maxAvatarUploadBytes+1)
	r.Error(err)
	assert.Equal(t, ftag.InvalidArgument, ftag.Get(err))
	r.Nil(storage.written)
}

// TestSet_ReEncodesValidImageAsPNG covers the happy path: a real image in a
// non-PNG format is decoded and re-encoded, matching what Get always claims
// to serve (image/png).
func TestSet_ReEncodesValidImageAsPNG(t *testing.T) {
	r := require.New(t)

	storage := &capturingStorer{}
	s := &service{storage: storage}

	src := image.NewRGBA(image.Rect(0, 0, 16, 16))
	src.Set(1, 1, color.RGBA{G: 255, A: 255})

	var jpegBuf bytes.Buffer
	r.NoError(jpeg.Encode(&jpegBuf, src, nil))

	err := s.Set(context.Background(), account.AccountID{}, &jpegBuf, int64(jpegBuf.Len()))
	r.NoError(err)

	r.NotEmpty(storage.written)
	img, err := png.Decode(bytes.NewReader(storage.written))
	r.NoError(err, "whatever was written must be a valid, decodable PNG regardless of the source format")
	r.Equal(16, img.Bounds().Dx())
}
