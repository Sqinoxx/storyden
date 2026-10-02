import { filter, flow, map } from "lodash/fp";

import { handle } from "@/api/client";
import {
  notificationUpdate,
  useNotificationList,
  useNotificationUpdateMany,
} from "@/api/openapi-client/notifications";
import {
  Notification,
  NotificationListResult,
  NotificationStatus,
} from "@/api/openapi-schema";
import { getCommonProperties } from "@/lib/datagraph/item";
import { Translations, useTranslation } from "@/lib/i18n";

import { NotificationItem } from "./item";

export type Props = {
  initialData?: NotificationListResult;
  status: NotificationStatus;
};

export function useNotifications(props: Props) {
  const t = useTranslation();
  const filterByStatus = filterStatus(props.status);
  const processNotifications = flow(
    filterByStatus,
    map((n: Notification) => mapToItem(t, n)),
  );

  const { data, error, mutate } = useNotificationList(
    { status: [props.status] },
    {
      swr: {
        fallbackData: props.initialData,
        revalidateIfStale: true,
        revalidateOnReconnect: true,
      },
    },
  );

  const markAllReadMutation = useNotificationUpdateMany();

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  const unreads = filterUnread(data.notifications).length;

  const notifications = processNotifications(data.notifications);

  async function handleMarkAs(id: string, status: NotificationStatus) {
    handle(async () => {
      await notificationUpdate(id, { status });

      if (data) {
        const newList = {
          ...data,
          notifications: data.notifications.map((n) => {
            if (n.id === id) {
              return { ...n, status };
            }
            return n;
          }),
        };

        await mutate(newList);
      }
    });
  }

  async function handleMarkAllAsRead() {
    handle(async () => {
      if (!data) {
        return;
      }

      const updated = data.notifications.map((n) => ({
        id: n.id,
        status: NotificationStatus.read,
      }));

      await markAllReadMutation.trigger({ notifications: updated });

      if (data) {
        const newList = {
          ...data,
          notifications: data.notifications.map((n) => ({
            ...n,
            status: "read" as NotificationStatus,
          })),
        };

        await mutate(newList);
      }
    });
  }

  return {
    ready: true as const,
    data: {
      unreads,
      notifications,
    },
    handlers: {
      handleMarkAs,
      handleMarkAllAsRead,
    },
  };
}

const filterStatus = (s: NotificationStatus) =>
  filter<Notification>((n) => n.status === s);

const filterUnread = filterStatus("unread");

function mapToItem(t: Translations, n: Notification): NotificationItem {
  const content = getNotificationContent(t, n);
  const createdAt = new Date(n.created_at);
  const title = n.source?.handle ?? t.notifications.system;
  const isRead = n.status === "read";

  return {
    id: n.id,
    createdAt,
    title,
    description: content.description,
    url: content.url,
    isRead,
    source: n.source,
    item: n.item,
  };
}

function getNotificationContent(t: Translations, n: Notification) {
  const p = n.item && getCommonProperties(n.item);
  const d = t.notifications.events;
  switch (n.event) {
    case "thread_reply":
      return { description: d.threadReply, url: `/t/locate/${p?.id}` };
    case "reply_to_reply":
      return { description: d.replyToReply, url: `/t/locate/${p?.id}` };
    case "post_like":
      return { description: d.postLike, url: `/t/locate/${p?.id}` };
    case "follow":
      return { description: d.follow, url: `/m/${n.source?.handle}` };
    case "profile_mention":
      return { description: d.profileMention, url: `/t/locate/${p?.id}` };
    case "event_host_added":
      return { description: d.eventHostAdded, url: `#` }; // not implemented
    case "member_attending_event":
      return { description: d.memberAttendingEvent, url: `#` }; // not implemented
    case "member_declined_event":
      return { description: d.memberDeclinedEvent, url: `#` }; // not implemented
    case "attendee_removed":
      return { description: d.attendeeRemoved, url: `#` }; // not implemented
    case "report_submitted":
      return { description: d.reportSubmitted, url: `/reports` };
    case "report_updated":
      return { description: d.reportUpdated, url: `/reports` };
    case "warning_issued":
      return {
        description: d.warningIssued,
        url: p?.slug ? `/m/${p.slug}` : "#",
      };
    case "node_version_created":
      return {
        description: d.nodeVersionCreated,
        url: p?.slug ? `/l/${p.slug}` : "#",
      };
    case "node_version_applied":
      return {
        description: d.nodeVersionApplied,
        url: p?.slug ? `/l/${p.slug}` : "#",
      };
    case "node_version_deleted":
      return {
        description: d.nodeVersionDeleted,
        url: p?.slug ? `/l/${p.slug}` : "#",
      };
  }
}
