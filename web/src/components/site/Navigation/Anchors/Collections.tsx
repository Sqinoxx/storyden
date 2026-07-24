"use client";

import { CollectionIcon } from "@/components/ui/icons/Collection";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const CollectionsID = "collections";
export const CollectionsRoute = "/c";
export const CollectionsLabel = "Collections";

export function CollectionsAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={CollectionsID}
      route={CollectionsRoute}
      label={t.nav.collections}
      icon={<CollectionIcon />}
      {...props}
    />
  );
}

export function CollectionsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={CollectionsID}
      route={CollectionsRoute}
      label={t.nav.collections}
      icon={<CollectionIcon />}
    />
  );
}
