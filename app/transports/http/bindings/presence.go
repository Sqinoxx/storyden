package bindings

import (
	"context"

	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"

	"github.com/Southclaws/storyden/app/resources/post/category"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/services/presence"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type Presence struct {
	tracker *presence.Tracker
}

func NewPresence(tracker *presence.Tracker) Presence {
	return Presence{tracker: tracker}
}

func (h *Presence) PresenceHeartbeat(ctx context.Context, request openapi.PresenceHeartbeatRequestObject) (openapi.PresenceHeartbeatResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	categoryID := opt.NewEmpty[category.CategoryID]()
	if request.Body != nil && request.Body.CategoryId != nil {
		categoryID = opt.New(category.CategoryID(openapi.ParseID(*request.Body.CategoryId)))
	}

	counts, err := h.tracker.Heartbeat(ctx, accountID, categoryID)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.PresenceHeartbeat200JSONResponse{
		PresenceHeartbeatOKJSONResponse: openapi.PresenceHeartbeatOKJSONResponse{
			Online:     counts.Online,
			InCategory: counts.InCategory.Ptr(),
		},
	}, nil
}
