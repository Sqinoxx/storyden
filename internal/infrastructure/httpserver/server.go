package httpserver

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"time"

	"go.uber.org/fx"

	"github.com/Southclaws/storyden/internal/boot_time"
	"github.com/Southclaws/storyden/internal/config"
)

func NewServer(lc fx.Lifecycle, logger *slog.Logger, cfg config.Config, router *http.ServeMux) *http.Server {
	server := &http.Server{
		Handler: router,
		Addr:    cfg.ListenAddr,

		// No ReadTimeout/WriteTimeout: uploads and SSE responses legitimately
		// run far longer than any fixed deadline would allow. ReadHeaderTimeout
		// alone is enough to stop a client from holding a connection open by
		// trickling headers one byte at a time (Slowloris), and IdleTimeout
		// bounds how long an idle keep-alive connection can be held.
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	wctx, cancel := context.WithCancel(context.Background())

	lc.Append(fx.Hook{
		OnStart: func(_ context.Context) error {
			server.BaseContext = func(ln net.Listener) context.Context { return wctx }
			go func() {
				// The HTTP server is the root node of dependency tree, because
				// it depends on everything else being initialised first. So, if
				// the app reaches this point, its considered a successful boot!

				logger.Info("storyden http server starting",
					slog.String("boot_time", time.Since(boot_time.StartedAt).String()),
					slog.String("address", cfg.ListenAddr),
					slog.String("api_address", cfg.PublicAPIAddress.String()),
					slog.String("web_address", cfg.PublicWebAddress.String()),
					slog.String("log_level", cfg.LogLevel.String()),
				)

				if err := server.ListenAndServe(); err != nil {
					logger.Error("http server stopped unexpectedly", slog.String("error", err.Error()))
					os.Exit(1)
				}
			}()
			return nil
		},
		OnStop: func(_ context.Context) error {
			cancel()
			return nil
		},
	})

	return server
}
