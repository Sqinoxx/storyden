"use client";

import { AdminIcon } from "@/components/ui/icons/Admin";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const AdminID = "admin";
export const AdminRoute = "/admin";
export const AdminLabel = "Admin";

type Props = AnchorProps & LinkButtonStyleProps;

export function AdminAnchor(props: Props) {
  const t = useTranslation();
  return (
    <Anchor
      id={AdminID}
      route={AdminRoute}
      label={t.nav.admin}
      icon={<AdminIcon />}
      {...props}
    />
  );
}

export function AdminMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={AdminID}
      route={AdminRoute}
      label={t.nav.admin}
      icon={<AdminIcon />}
    />
  );
}
