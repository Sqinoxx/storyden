package bindings

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/fault/ftag"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/pagination"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

func similarList(items datagraph.ItemList) openapi.ThreadSimilarList {
	return openapi.ThreadSimilarList{Items: dt.Map(items, serialiseDatagraphItem)}
}

func (i *Threads) ThreadSimilar(ctx context.Context, request openapi.ThreadSimilarRequestObject) (openapi.ThreadSimilarResponseObject, error) {
	body := ""
	if raw := opt.NewPtr(request.Body.Body).OrZero(); raw != "" {
		content, err := datagraph.NewRichText(raw)
		if err != nil {
			return nil, fault.Wrap(err, fctx.With(ctx), ftag.With(ftag.InvalidArgument))
		}
		body = content.Plaintext()
	}

	items, err := i.similar.ForDraft(ctx, opt.NewPtr(request.Body.Title).OrZero(), body)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ThreadSimilar200JSONResponse{
		ThreadSimilarOKJSONResponse: openapi.ThreadSimilarOKJSONResponse(similarList(items)),
	}, nil
}

func (i *Threads) ThreadRelated(ctx context.Context, request openapi.ThreadRelatedRequestObject) (openapi.ThreadRelatedResponseObject, error) {
	postID, err := i.thread_mark_svc.Lookup(ctx, string(request.ThreadMark))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	thr, err := i.thread_svc.Get(ctx, postID, pagination.NewPageParams(1, 1))
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}
	if thr.Access.Ok() {
		return nil, fault.New("thread not found", fctx.With(ctx), ftag.With(ftag.NotFound))
	}

	items, err := i.similar.Related(ctx, postID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.ThreadRelated200JSONResponse{
		ThreadSimilarOKJSONResponse: openapi.ThreadSimilarOKJSONResponse(similarList(items)),
	}, nil
}
