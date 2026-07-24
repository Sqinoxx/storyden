"use client";

import { HomeIcon } from "@/components/ui/icons/Home";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const HomeID = "home";
export const HomeRoute = "/d";
export const HomeLabel = "Home";

export function HomeAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={HomeID}
      route={HomeRoute}
      label={t.nav.home}
      icon={<HomeIcon />}
      {...props}
    />
  );
}

export function HomeMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={HomeID}
      route={HomeRoute}
      label={t.nav.home}
      icon={<HomeIcon />}
    />
  );
}
