package openapi

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/fmsg"
	"github.com/Southclaws/fault/ftag"
	"github.com/labstack/echo/v4"
)

func TestHTTPErrorHandlerWritesProblemDetails(t *testing.T) {
	ctx := fctx.WithMeta(context.Background(), "code", "validation_error")
	err := fault.New("invalid account payload",
		fctx.With(ctx),
		fmsg.WithDesc("invalid account", "The account payload is invalid."),
		ftag.With(ftag.InvalidArgument),
	)

	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/", nil).WithContext(ctx)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := HTTPErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))
	handler(err, c)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}

	contentType := rec.Header().Get(echo.HeaderContentType)
	if !strings.HasPrefix(contentType, ProblemJSONMediaType) {
		t.Fatalf("expected content type %q, got %q", ProblemJSONMediaType, contentType)
	}

	var problem APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal problem details: %v", err)
	}

	if problem.Title == nil || *problem.Title != "The account payload is invalid." {
		t.Fatalf("expected fmsg description as title, got %#v", problem.Title)
	}

	if problem.Detail == nil || *problem.Detail != err.Error() {
		t.Fatalf("expected err.Error as detail, got %#v", problem.Detail)
	}

	if problem.Type == nil || *problem.Type != "urn:storyden:problem:invalid-argument" {
		t.Fatalf("expected problem type from ftag kind, got %#v", problem.Type)
	}

	if problem.TraceId == "" {
		t.Fatal("expected trace_id")
	}
}

func problemFor(t *testing.T, err error) APIError {
	t.Helper()

	e := echo.New()
	rec := httptest.NewRecorder()
	c := e.NewContext(httptest.NewRequest(http.MethodGet, "/", nil), rec)

	HTTPErrorHandler(slog.New(slog.NewTextHandler(io.Discard, nil)))(err, c)

	var problem APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &problem); err != nil {
		t.Fatalf("failed to unmarshal problem details: %v", err)
	}

	return problem
}

func TestHTTPErrorHandlerHidesInternalMetadata(t *testing.T) {
	ctx := fctx.WithMeta(context.Background(),
		"location", "/app/services/thing.go:42",
		"package", "thing",
		"client.address", "203.0.113.7",
		"http.request.method", "GET",
		"url.query", "a=b",
		"asset_id", "abc",
	)

	problem := problemFor(t, fault.New("nope", fctx.With(ctx), ftag.With(ftag.NotFound)))

	if problem.Metadata == nil {
		t.Fatal("expected public metadata to remain")
	}

	meta := *problem.Metadata
	if meta["asset_id"] != "abc" {
		t.Fatalf("expected asset_id to remain, got %#v", meta)
	}

	for _, k := range []string{"location", "package", "client.address", "http.request.method", "url.query"} {
		if _, ok := meta[k]; ok {
			t.Fatalf("internal key %q leaked: %#v", k, meta)
		}
	}
}

func TestHTTPErrorHandlerHidesInternalErrorDetail(t *testing.T) {
	ctx := fctx.WithMeta(context.Background(), "query", "select secret")

	problem := problemFor(t, fault.New("pq: relation does not exist", fctx.With(ctx)))

	if problem.Detail != nil {
		t.Fatalf("expected no detail on internal errors, got %q", *problem.Detail)
	}

	if problem.Metadata != nil {
		t.Fatalf("expected no metadata on internal errors, got %#v", *problem.Metadata)
	}

	if problem.TraceId == "" {
		t.Fatal("expected trace_id")
	}
}
