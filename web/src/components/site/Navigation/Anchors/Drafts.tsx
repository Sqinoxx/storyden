"use client";

import { DraftIcon } from "@/components/ui/icons/Draft";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const DraftsID = "drafts";
export const DraftsRoute = "/drafts";
export const DraftsLabel = "Drafts";
export const DraftsIcon = <DraftIcon />;

export function DraftsAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={DraftsID}
      route={DraftsRoute}
      label={t.nav.drafts}
      icon={DraftsIcon}
      {...props}
    />
  );
}

export function DraftsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={DraftsID}
      route={DraftsRoute}
      label={t.nav.drafts}
      icon={DraftsIcon}
    />
  );
}
