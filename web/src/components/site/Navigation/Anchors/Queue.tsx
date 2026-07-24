"use client";

import { QueueIcon } from "@/components/ui/icons/Queue";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const QueueID = "queue";
export const QueueRoute = "/queue";
export const QueueLabel = "Queue";

export function QueueAnchor(props: LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={QueueID}
      route={QueueRoute}
      label={t.nav.queue}
      icon={<QueueIcon />}
      {...props}
    />
  );
}

export function QueueMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={QueueID}
      route={QueueRoute}
      label={t.nav.queue}
      icon={<QueueIcon />}
    />
  );
}
