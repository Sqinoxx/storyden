"use client";

import { CreateIcon } from "@/components/ui/icons/Create";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const ComposeID = "compose";
export const ComposeRoute = "/new";
export const ComposeLabel = "Post";
export const ComposeIcon = <CreateIcon />;

export function ComposeAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={ComposeID}
      route={ComposeRoute}
      label={t.nav.compose}
      icon={ComposeIcon}
      {...props}
    />
  );
}

export function ComposeMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={ComposeID}
      route={ComposeRoute}
      label={t.nav.compose}
      icon={ComposeIcon}
    />
  );
}
