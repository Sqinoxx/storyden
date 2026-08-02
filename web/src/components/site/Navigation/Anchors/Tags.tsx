"use client";

import { TagIcon } from "@/components/ui/icons/Tag";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";

export const TagsID = "tags";
export const TagsRoute = "/admin/tags";

export function TagsAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={TagsID}
      route={TagsRoute}
      label={t.nav.tags}
      icon={<TagIcon />}
      {...props}
    />
  );
}

export function TagsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={TagsID}
      route={TagsRoute}
      label={t.nav.tags}
      icon={<TagIcon />}
    />
  );
}
