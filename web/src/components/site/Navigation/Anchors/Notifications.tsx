"use client";

import { NotificationIcon } from "@/components/ui/icons/Notification";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const NotificationsID = "notifications";
export const NotificationsRoute = "/notifications";
export const NotificationsLabel = "Notifications";
export const NotificationsIcon = <NotificationIcon />;

export function NotificationsAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={NotificationsID}
      route={NotificationsRoute}
      label={t.nav.notifications}
      icon={NotificationsIcon}
      {...props}
    />
  );
}

export function NotificationsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={NotificationsID}
      route={NotificationsRoute}
      label={t.nav.notifications}
      icon={NotificationsIcon}
    />
  );
}
