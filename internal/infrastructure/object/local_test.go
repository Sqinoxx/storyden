package object

import (
	"context"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Southclaws/storyden/internal/config"
)

func TestLocalStorerListMissingPrefixReturnsEmpty(t *testing.T) {
	t.Parallel()

	store := NewLocalStorer(config.Config{
		AssetStorageLocalPath: t.TempDir(),
	})

	got, err := store.List(context.Background(), "plugins")
	require.NoError(t, err)
	require.Empty(t, got)
}

func TestLocalStorerAcceptsOSNativeNestedPaths(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := NewLocalStorer(config.Config{
		AssetStorageLocalPath: t.TempDir(),
	})

	path := filepath.Join("plugins", "nested", "archive.zip")
	require.NoError(t, store.Write(ctx, path, strings.NewReader("data"), 4))

	exists, err := store.Exists(ctx, path)
	require.NoError(t, err)
	require.True(t, exists)

	r, size, err := store.Read(ctx, path)
	require.NoError(t, err)
	require.EqualValues(t, 4, size)
	b, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, "data", string(b))
	if c, ok := r.(io.Closer); ok {
		c.Close()
	}

	names, err := store.List(ctx, filepath.Join("plugins", "nested"))
	require.NoError(t, err)
	require.Equal(t, []string{"archive.zip"}, names)
}
