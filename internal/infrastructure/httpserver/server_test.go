package httpserver

import (
	"log/slog"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/fx/fxtest"

	"github.com/Southclaws/storyden/internal/config"
)

// TestNewServerSetsTimeouts covers B15: the server had no header/idle
// timeouts at all, so a client trickling bytes (or just holding a keep-alive
// connection open) could tie up a listener slot indefinitely.
func TestNewServerSetsTimeouts(t *testing.T) {
	t.Parallel()

	logger := slog.New(slog.DiscardHandler)

	server := NewServer(fxtest.NewLifecycle(t), logger, config.Config{}, http.NewServeMux())

	assert.Equal(t, 10*time.Second, server.ReadHeaderTimeout)
	assert.Equal(t, 120*time.Second, server.IdleTimeout)
	assert.Equal(t, 1<<20, server.MaxHeaderBytes)

	// Deliberately unset: uploads and SSE responses run far longer than any
	// fixed deadline would allow.
	assert.Zero(t, server.ReadTimeout)
	assert.Zero(t, server.WriteTimeout)
}
