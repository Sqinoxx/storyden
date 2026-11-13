package bindings

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"

	"github.com/Southclaws/storyden/app/resources/asset"
	"github.com/Southclaws/storyden/app/resources/settings"
	"github.com/Southclaws/storyden/app/services/pdf_summary"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func (i *Threads) requirePDFSummary(ctx context.Context) error {
	enabled, err := i.aiEnabled(ctx, (*settings.Settings).AIPDFSummaryEnabled)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}
	if !enabled {
		return fault.Wrap(errAIFeatureDisabled, fctx.With(ctx))
	}
	return nil
}

func (i *Threads) ThreadAttachmentSummaryGet(ctx context.Context, request openapi.ThreadAttachmentSummaryGetRequestObject) (openapi.ThreadAttachmentSummaryGetResponseObject, error) {
	if err := i.requirePDFSummary(ctx); err != nil {
		return nil, err
	}

	postID, err := i.thread_mark_svc.Lookup(ctx, string(request.ThreadMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	res, err := i.pdfSummary.Get(ctx, postID, asset.AssetID(openapi.ParseID(request.AssetId)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	out := openapi.AttachmentSummary{
		Status:  openapi.AttachmentSummaryStatus(res.Status),
		Preview: res.Preview,
	}
	if res.Status == pdf_summary.StatusReady {
		out.Summary = &res.Summary
		q := openapi.AttachmentSummaryQuality(res.Quality)
		out.Quality = &q
	}

	return openapi.ThreadAttachmentSummaryGet200JSONResponse{
		ThreadAttachmentSummaryGetOKJSONResponse: openapi.ThreadAttachmentSummaryGetOKJSONResponse(out),
	}, nil
}

func (i *Threads) ThreadAttachmentSummaryStream(ctx context.Context, request openapi.ThreadAttachmentSummaryStreamRequestObject) (openapi.ThreadAttachmentSummaryStreamResponseObject, error) {
	if err := i.requirePDFSummary(ctx); err != nil {
		return nil, err
	}

	postID, err := i.thread_mark_svc.Lookup(ctx, string(request.ThreadMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	events, err := i.pdfSummary.Stream(ctx, postID, asset.AssetID(openapi.ParseID(request.AssetId)))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return attachmentSummaryStream{ctx: ctx, events: events}, nil
}

type attachmentSummaryStream struct {
	ctx    context.Context
	events func(yield func(pdf_summary.Event) bool)
}

func (s attachmentSummaryStream) VisitThreadAttachmentSummaryStreamResponse(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, _ := w.(http.Flusher)
	if flusher == nil {
		if inner := unwrapWriter(w); inner != nil {
			flusher, _ = inner.(http.Flusher)
		}
	}

	w.WriteHeader(http.StatusOK)

	send := func(event string, payload any) bool {
		data, err := json.Marshal(payload)
		if err != nil {
			return false
		}
		if event != "" {
			if _, err := fmt.Fprintf(w, "event: %s\n", event); err != nil {
				return false
			}
		}
		if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
			return false
		}
		if flusher != nil {
			flusher.Flush()
		}
		return s.ctx.Err() == nil
	}

	for ev := range s.events {
		switch {
		case ev.Done && ev.Err != nil:
			send("error", map[string]string{"message": "summary failed"})
			return nil
		case ev.Done:
			send("done", map[string]string{"summary": ev.HTML})
			return nil
		default:
			if !send("", map[string]string{"t": ev.Text}) {
				return nil
			}
		}
	}
	return nil
}
