"use client";

import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";

export const MarketplaceID = "marketplace";
export const MarketplaceRoute = "/flohmarkt";

export function MarketplaceAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={MarketplaceID}
      route={MarketplaceRoute}
      label={t.nav.marketplace}
      icon={<MarketplaceIcon />}
      {...props}
    />
  );
}

export function MarketplaceMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={MarketplaceID}
      route={MarketplaceRoute}
      label={t.nav.marketplace}
      icon={<MarketplaceIcon />}
    />
  );
}
