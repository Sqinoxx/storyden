"use client";

import { ProfileIcon } from "@/components/ui/icons/Profile";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const ProfileID = "profile";
export const ProfileRoute = (handle: string) => `/m/${handle}`;
export const ProfileLabel = "Profile";

export function ProfileAnchor({
  handle,
  ...props
}: LinkButtonStyleProps & { handle: string }) {
  const t = useTranslation();
  return (
    <Anchor
      id={ProfileID}
      route={ProfileRoute(handle)}
      label={t.nav.profile}
      icon={<ProfileIcon />}
      {...props}
    />
  );
}

export function ProfileMenuItem({ handle }: { handle: string }) {
  const t = useTranslation();
  return (
    <MenuItem
      id={ProfileID}
      route={ProfileRoute(handle)}
      label={t.nav.profile}
      icon={<ProfileIcon />}
    />
  );
}
