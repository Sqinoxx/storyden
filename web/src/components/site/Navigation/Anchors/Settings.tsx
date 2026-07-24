"use client";

import { SettingsIcon } from "@/components/ui/icons/Settings";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const SettingsID = "settings";
export const SettingsRoute = "/settings";
export const SettingsLabel = "Settings";

type Props = AnchorProps & LinkButtonStyleProps;

export function SettingsAnchor(props: Props) {
  const t = useTranslation();
  return (
    <Anchor
      id={SettingsID}
      route={SettingsRoute}
      label={t.nav.settings}
      icon={<SettingsIcon />}
      {...props}
    />
  );
}

export function SettingsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={SettingsID}
      route={SettingsRoute}
      label={t.nav.settings}
      icon={<SettingsIcon />}
    />
  );
}
