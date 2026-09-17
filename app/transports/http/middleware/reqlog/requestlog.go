package reqlog

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"runtime/debug"
	"time"

	"github.com/Southclaws/storyden/app/services/reqinfo"
	"github.com/Southclaws/storyden/app/transports/http/middleware/origin"
	"github.com/Southclaws/storyden/internal/infrastructure/instrumentation/kv"
	"github.com/Southclaws/storyden/internal/infrastructure/instrumentation/spanner"
)

const redacted = "[redacted]"

// redactedQueryKeys lists query parameters that carry a bearer-equivalent
// secret (OAuth codes/state/tokens, password reset/verification tokens): a
// value that, on its own, lets whoever reads the logs act as the request's
// caller. Logs are often shipped to, retained by, and accessed through
// systems with a much wider audience than the app or its database, so these
// must never appear in them even though the request itself is legitimate.
var redactedQueryKeys = map[string]bool{
	"code":          true,
	"state":         true,
	"token":         true,
	"user_code":     true,
	"access_token":  true,
	"refresh_token": true,
}

// redactedQuery renders a URL's query string with sensitive values replaced,
// preserving every key (including repeated ones) so the shape of the request
// is still visible for debugging.
func redactedQuery(u *url.URL) string {
	values := u.Query()

	for key := range values {
		if redactedQueryKeys[key] {
			for i := range values[key] {
				values[key][i] = redacted
			}
		}
	}

	return values.Encode()
}

type Middleware struct {
	ins spanner.Instrumentation
}

func New(ins spanner.Builder) *Middleware {
	return &Middleware{
		ins: ins.Build(),
	}
}

type withStatus struct {
	http.ResponseWriter
	statusCode int
}

func (w *withStatus) Unwrap() http.ResponseWriter {
	return w.ResponseWriter
}

func (lrw *withStatus) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}

func (m *Middleware) WithLogger() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			origin := origin.GetOrigin(r.Context())

			// log entries should be in the form "GET /a/b/c".
			title := r.Method + " " + r.URL.Path

			wr := &withStatus{ResponseWriter: w}
			clientAddress := reqinfo.GetClientAddress(r.Context())
			if clientAddress == "" {
				clientAddress = r.RemoteAddr
			}

			ctx, span := m.ins.InstrumentNamed(r.Context(), title,
				kv.String("http.request.header.origin", origin),
				kv.String("client.address", clientAddress),
				kv.String("http.request.method", r.Method),
				kv.String("url.query", redactedQuery(r.URL)),
				kv.Int("http.request.body.size", int(r.ContentLength)),
			)
			defer span.End()

			defer func() {
				span.Annotate(
					kv.Duration("duration", time.Since(start)),
					kv.Int("http.response.status_code", wr.statusCode),
				)

				logger := span.Logger()

				logger.Info(title)

				if recovery := recover(); recovery != nil {
					err := func(v any) error {
						if e, ok := v.(error); ok {
							return e
						} else {
							return fmt.Errorf("%v", v)
						}
					}(recovery)

					trace := debug.Stack()

					errorlog := title + ": " + err.Error()

					err = span.Wrap(err, errorlog)

					logger.Error(errorlog,
						slog.String("error", err.Error()),
						slog.Any("trace", trace),
					)

					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}()

			next.ServeHTTP(wr, r.WithContext(ctx))
		})
	}
}
