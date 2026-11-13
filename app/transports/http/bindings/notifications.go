package bindings

import (
	"context"

	"github.com/Southclaws/dt"
	"github.com/Southclaws/fault"
	"github.com/Southclaws/fault/fctx"
	"github.com/Southclaws/opt"
	"github.com/rs/xid"

	"github.com/Southclaws/storyden/app/resources/account/notification"
	"github.com/Southclaws/storyden/app/resources/account/notification/notify_querier"
	"github.com/Southclaws/storyden/app/resources/account/notification/notify_writer"
	"github.com/Southclaws/storyden/app/resources/datagraph"
	"github.com/Southclaws/storyden/app/resources/link/link_ref"
	"github.com/Southclaws/storyden/app/resources/post"
	"github.com/Southclaws/storyden/app/services/access/access_check"
	"github.com/Southclaws/storyden/app/services/authentication/session"
	"github.com/Southclaws/storyden/app/transports/http/openapi"
)

type Notifications struct {
	notifyReader *notify_querier.Querier
	notifyWriter *notify_writer.Writer
	access       *access_check.Checker
}

func NewNotifications(
	notifyReader *notify_querier.Querier,
	notifyWriter *notify_writer.Writer,
	access *access_check.Checker,
) Notifications {
	return Notifications{
		notifyReader: notifyReader,
		notifyWriter: notifyWriter,
		access:       access,
	}
}

// withhold keeps notifications about locked posts, so the member still learns
// that something happened and can follow the link, but drops their text.
func (h *Notifications) withhold(ctx context.Context, list notification.Notifications) error {
	ids := []xid.ID{}
	for _, n := range list {
		if n != nil && n.Item != nil {
			ids = append(ids, n.Item.GetID())
		}
	}

	locked, err := h.access.LockedPosts(ctx, ids...)
	if err != nil {
		return fault.Wrap(err, fctx.With(ctx))
	}

	for _, n := range list {
		if n == nil || n.Item == nil {
			continue
		}
		if _, ok := locked[n.Item.GetID()]; !ok {
			continue
		}
		if p, isPost := n.Item.(*post.Post); isPost {
			p.Content = datagraph.Content{}
			p.Assets = nil
			p.WebLink = opt.NewEmpty[link_ref.LinkRef]()
		}
	}

	return nil
}

func (h *Notifications) NotificationList(ctx context.Context, request openapi.NotificationListRequestObject) (openapi.NotificationListResponseObject, error) {
	session, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	list, err := h.notifyReader.ListNotifications(ctx, session)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if err := h.withhold(ctx, list); err != nil {
		return nil, err
	}

	notifications := dt.Map(list, serialiseNotification)

	return openapi.NotificationList200JSONResponse{
		NotificationListOKJSONResponse: openapi.NotificationListOKJSONResponse{
			// TODO: Pagination at resource layer.
			CurrentPage:   1,
			NextPage:      new(int),
			Notifications: notifications,
			PageSize:      len(notifications),
			Results:       len(notifications),
			TotalPages:    1,
		},
	}, nil
}

func (h *Notifications) NotificationUpdate(ctx context.Context, request openapi.NotificationUpdateRequestObject) (openapi.NotificationUpdateResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	if request.Body.Status == nil {
		return nil, nil
	}

	id := openapi.ParseID(request.NotificationId)

	status := opt.Map(opt.NewPtr(request.Body.Status), deserialiseNotificationStatus)

	n, err := h.notifyWriter.SetRead(ctx, accountID, id, status.OrZero())
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	return openapi.NotificationUpdate200JSONResponse{
		NotificationUpdateOKJSONResponse: openapi.NotificationUpdateOKJSONResponse(serialiseNotificationRef(n)),
	}, nil
}

func (h *Notifications) NotificationUpdateMany(ctx context.Context, request openapi.NotificationUpdateManyRequestObject) (openapi.NotificationUpdateManyResponseObject, error) {
	accountID, err := session.GetAccountID(ctx)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	notificationRefs := make([]*notification.NotificationRef, 0, len(request.Body.Notifications))
	for _, n := range request.Body.Notifications {
		if n.Status == nil {
			continue
		}

		notificationRefs = append(notificationRefs, &notification.NotificationRef{
			ID:   openapi.ParseID(n.Id),
			Read: deserialiseNotificationStatus(*n.Status),
		})
	}

	updated, err := h.notifyWriter.UpdateStatusMany(ctx, accountID, notificationRefs)
	if err != nil {
		return nil, fault.Wrap(err, fctx.With(ctx))
	}

	notifications := dt.Map(updated, serialiseNotificationRef)

	return openapi.NotificationUpdateMany200JSONResponse{
		NotificationUpdateManyOKJSONResponse: openapi.NotificationUpdateManyOKJSONResponse{
			CurrentPage:   1,
			NextPage:      new(int),
			Notifications: notifications,
			PageSize:      len(notifications),
			Results:       len(notifications),
			TotalPages:    1,
		},
	}, nil
}

func serialiseNotification(in *notification.Notification) openapi.Notification {
	item := opt.Map(opt.NewSafe(in.Item, in.Item != nil), serialiseDatagraphItem)

	return openapi.Notification{
		Id:        in.ID.String(),
		CreatedAt: in.Time,
		Event:     openapi.NotificationEvent(in.Event.String()),
		Item:      item.Ptr(),
		Source:    opt.Map(in.Source, serialiseProfileReference).Ptr(),
		Status:    serialiseNotificationStatus(in.Read),
	}
}

func serialiseNotificationRef(in *notification.NotificationRef) openapi.Notification {
	return openapi.Notification{
		Id:        in.ID.String(),
		CreatedAt: in.Time,
		Event:     openapi.NotificationEvent(in.Event.String()),
		Source:    opt.Map(in.Source, serialiseProfileReference).Ptr(),
		Status:    serialiseNotificationStatus(in.Read),
	}
}

func serialiseNotificationStatus(in bool) openapi.NotificationStatus {
	if in {
		return openapi.Read
	}
	return openapi.Unread
}

func deserialiseNotificationStatus(in openapi.NotificationStatus) bool {
	return in == openapi.Read
}
