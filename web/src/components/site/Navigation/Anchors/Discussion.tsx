"use client";

import { DiscussionIcon } from "@/components/ui/icons/Discussion";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const DiscussionID = "discussion";
export const DiscussionRoute = "/d";
export const DiscussionLabel = "Discussion";

export function DiscussionAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={DiscussionID}
      route={DiscussionRoute}
      label={t.nav.discussion}
      icon={<DiscussionIcon />}
      {...props}
    />
  );
}

export function DiscussionMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={DiscussionID}
      route={DiscussionRoute}
      label={t.nav.discussion}
      icon={<DiscussionIcon />}
    />
  );
}
