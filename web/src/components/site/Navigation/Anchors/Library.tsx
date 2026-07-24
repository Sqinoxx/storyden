"use client";

import { LibraryIcon } from "@/components/ui/icons/Library";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const LibraryID = "library";
export const LibraryRoute = "/l";
export const LibraryLabel = "Library";

export function LibraryAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={LibraryID}
      route={LibraryRoute}
      label={t.nav.library}
      icon={<LibraryIcon />}
      {...props}
    />
  );
}

export function LibraryMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={LibraryID}
      route={LibraryRoute}
      label={t.nav.library}
      icon={<LibraryIcon />}
    />
  );
}
