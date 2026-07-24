"use client";

import { RolesIcon } from "@/components/ui/icons/Roles";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const RolesID = "roles";
export const RolesRoute = "/roles";
export const RolesLabel = "Roles";

export function RolesAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={RolesID}
      route={RolesRoute}
      label={t.nav.roles}
      icon={<RolesIcon />}
      {...props}
    />
  );
}

export function RolesMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={RolesID}
      route={RolesRoute}
      label={t.nav.roles}
      icon={<RolesIcon />}
    />
  );
}
