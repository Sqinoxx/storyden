package ocr

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

type fakeTextLayer struct {
	text  string
	err   error
	calls int
}

func (f *fakeTextLayer) Available() bool { return true }
func (f *fakeTextLayer) Extract(ctx context.Context, pdfPath string) (string, int, error) {
	f.calls++
	if _, err := os.Stat(pdfPath); err != nil {
		return "", 0, err
	}
	return f.text, 1, f.err
}

func TestLayeredClient_PrefersIsolatedTextLayer(t *testing.T) {
	tl := &fakeTextLayer{text: "Klausur Prothetik Wintersemester mit Loesungen und Kommentaren"}
	c := NewLayeredClient(slog.Default(), nil, nil, tl, config.Config{})

	result, err := c.ExtractText(context.Background(), []byte("%PDF-1.4 not parsed in process"), "application/pdf")
	require.NoError(t, err)
	assert.Equal(t, 1, tl.calls)
	assert.Equal(t, "pdf-textlayer", result.Engine)
	assert.Contains(t, result.Text, "Prothetik")
}

func TestLayeredClient_IsolatedTextLayerFailureDoesNotFallBackInProcess(t *testing.T) {
	tl := &fakeTextLayer{err: errors.New("signal: killed")}
	c := NewLayeredClient(slog.Default(), nil, nil, tl, config.Config{})

	data := makeTextPDF(t, "Spanisch fuer Mediziner im vierten Semester ist Pflicht")

	_, err := c.ExtractText(context.Background(), data, "application/pdf")
	require.ErrorIs(t, err, ErrNoTextLayer)
}

func TestLayeredClient_CancelledTextLayerStops(t *testing.T) {
	tl := &fakeTextLayer{err: context.Canceled}
	c := NewLayeredClient(slog.Default(), nil, nil, tl, config.Config{})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := c.ExtractText(ctx, makeTextPDF(t, "egal"), "application/pdf")
	require.ErrorIs(t, err, context.Canceled)
}

func TestPopplerTextExtractor_ReadsTextLayer(t *testing.T) {
	p := NewPopplerTextExtractor("")
	if !p.Available() {
		t.Skip("pdftotext not installed")
	}

	path := filepath.Join(t.TempDir(), "in.pdf")
	require.NoError(t, os.WriteFile(path, makeTextPDF(t, "Auslandssemester Spanisch fuer Mediziner"), 0o600))

	text, pages, err := p.Extract(context.Background(), path)
	require.NoError(t, err)
	assert.Equal(t, 1, pages)
	assert.Contains(t, text, "Spanisch")
	assert.NotContains(t, text, "\f")
}

func TestPopplerTextExtractor_UnavailableReportsEngineUnavailable(t *testing.T) {
	p := NewPopplerTextExtractor("storyden-no-such-pdftotext")

	_, _, err := p.Extract(context.Background(), "in.pdf")
	require.ErrorIs(t, err, ErrEngineUnavailable)
}

func TestMemoryLimitedCommand_AppliesLimit(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("memory limit only applies on linux")
	}

	var out bytes.Buffer
	cmd := memoryLimitedCommand(context.Background(), 123456, "sh", "-c", "ulimit -v")
	cmd.Stdout = &out
	require.NoError(t, cmd.Run())
	assert.Equal(t, "123456", strings.TrimSpace(out.String()))
}

func TestCappedBuffer_DropsOverflow(t *testing.T) {
	b := cappedBuffer{limit: 5}

	n, err := b.Write([]byte("abc"))
	require.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = b.Write([]byte("defgh"))
	require.NoError(t, err)
	assert.Equal(t, 5, n)

	assert.Equal(t, "abcde", b.String())
}
