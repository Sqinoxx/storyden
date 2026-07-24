"use client";

import { SearchIcon } from "@/components/ui/icons/Search";
import { LinkButtonStyleProps } from "@/components/ui/link-button";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const SearchID = "search";
export const SearchRoute = "/search";
export const SearchLabel = "Search";

type Props = AnchorProps & LinkButtonStyleProps;

export function SearchAnchor(props: Props) {
  const t = useTranslation();
  return (
    <Anchor
      id={SearchID}
      route={SearchRoute}
      label={t.nav.search}
      icon={<SearchIcon />}
      {...props}
    />
  );
}

export function SearchMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={SearchID}
      route={SearchRoute}
      label={t.nav.search}
      icon={<SearchIcon />}
    />
  );
}
