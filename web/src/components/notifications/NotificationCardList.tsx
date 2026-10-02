import { handle } from "@/api/client";
import { NotificationStatus } from "@/api/openapi-schema";
import { ArchiveIcon } from "@/components/ui/icons/Archive";
import { InboxIcon } from "@/components/ui/icons/Inbox";
import { Card, CardRows } from "@/components/ui/rich-card";
import { getCommonProperties } from "@/lib/datagraph/item";
import { useLanguage, useTranslation } from "@/lib/i18n";
import { Center, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { timestamp } from "@/utils/date";

import { MemberBadge } from "../member/MemberBadge/MemberBadge";
import { Button } from "../ui/button";
import { IconButton } from "../ui/icon-button";

import { NotificationItem } from "./item";

type Props = {
  notifications: NotificationItem[];
  onMove: (id: string, status: NotificationStatus) => Promise<void>;
};

export function NotificationCardList({ notifications, onMove }: Props) {
  const { t, language } = useLanguage();
  if (notifications.length === 0) {
    return (
      <Center h="96" w="full" display="flex" flexDirection="column" gap="1">
        <styled.p color="fg.muted">{t.notifications.empty}</styled.p>
      </Center>
    );
  }

  return (
    <CardRows>
      {notifications.map((n) => {
        const properties = n.item && getCommonProperties(n.item);

        const title = properties?.description
          ? `${n.description} "${properties?.description}"`
          : n.description;

        return (
          <Card
            key={n.id}
            id={n.id}
            shape="row"
            title={timestamp(n.createdAt, false, language)}
            text={title}
            url={n.url}
            // controls={}
          >
            <WStack>
              <NotificationSource {...n} />
              <StatusControl notification={n} onMove={onMove} />
            </WStack>
          </Card>
        );
      })}
    </CardRows>
  );
}

function NotificationSource(props: NotificationItem) {
  const t = useTranslation();
  if (props.source) {
    return (
      <MemberBadge profile={props.source} size="sm" name="full-horizontal" />
    );
  }

  return (
    <HStack>
      <LStack gap="0">
        <styled.span color="fg.subtle">
          {t.notifications.systemMessage}
        </styled.span>
      </LStack>
    </HStack>
  );
}

function StatusControl({
  notification,
  onMove,
}: {
  notification: NotificationItem;
  onMove: (id: string, status: NotificationStatus) => void;
}) {
  const t = useTranslation();

  function handleChangeStatus() {
    handle(async () => {
      const newStatus = notification.isRead ? "unread" : "read";
      onMove(notification.id, newStatus);
    });
  }

  return notification.isRead ? (
    <IconButton
      variant="ghost"
      size="xs"
      title={t.notifications.markAsUnread}
      onClick={handleChangeStatus}
    >
      <InboxIcon color="fg.subtle" />
    </IconButton>
  ) : (
    <IconButton
      variant="ghost"
      size="xs"
      title={t.notifications.markAsRead}
      onClick={handleChangeStatus}
    >
      <ArchiveIcon color="fg.subtle" />
    </IconButton>
  );
}
