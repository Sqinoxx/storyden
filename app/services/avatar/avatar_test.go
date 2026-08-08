package avatar

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"io"
	"testing"

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
