"use client";

import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { isMarketplaceBeta } from "@/lib/marketplace/visibility";
import { useSettings } from "@/lib/settings/settings-client";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";

export const MarketplaceID = "marketplace";
export const MarketplaceRoute = "/flohmarkt";

function useMarketplaceLabel() {
  const t = useTranslation();
  const settings = useSettings(undefined, true);
  const beta =
    settings.ready && isMarketplaceBeta(settings.settings.marketplace_enabled);
  return beta
    ? `${t.nav.marketplace} (${t.marketSettings.betaBadge})`
    : t.nav.marketplace;
}

export function MarketplaceAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const label = useMarketplaceLabel();
  return (
    <Anchor
      id={MarketplaceID}
      route={MarketplaceRoute}
      label={label}
      icon={<MarketplaceIcon />}
      {...props}
    />
  );
}

export function MarketplaceMenuItem() {
  const label = useMarketplaceLabel();
  return (
    <MenuItem
      id={MarketplaceID}
      route={MarketplaceRoute}
      label={label}
      icon={<MarketplaceIcon />}
    />
  );
}
