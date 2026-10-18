"use client";

import { LatestIcon } from "@/components/ui/icons/Latest";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";

export const LatestID = "latest";
export const LatestRoute = "/d";

export function LatestAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={LatestID}
      route={LatestRoute}
      label={t.nav.latest}
      icon={<LatestIcon />}
      {...props}
    />
  );
}

export function LatestMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={LatestID}
      route={LatestRoute}
      label={t.nav.latest}
      icon={<LatestIcon />}
    />
  );
}
