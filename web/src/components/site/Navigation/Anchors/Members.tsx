"use client";

import { MembersIcon } from "@/components/ui/icons/Members";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const MembersID = "members";
export const MembersRoute = "/m";
export const MembersLabel = "Members";

export function MembersAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={MembersID}
      route={MembersRoute}
      label={t.nav.members}
      icon={<MembersIcon />}
      {...props}
    />
  );
}

export function MembersMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={MembersID}
      route={MembersRoute}
      label={t.nav.members}
      icon={<MembersIcon />}
    />
  );
}
