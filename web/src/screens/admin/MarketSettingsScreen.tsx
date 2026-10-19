"use client";

import { useAdminSettingsGet } from "@/api/openapi-client/admin";
import { UnreadyBanner } from "@/components/site/Unready";
import { parseAdminSettings } from "@/lib/settings/settings";

import { MarketSettingsForm } from "../../components/admin/MarketSettings/MarketSettings";

export function MarketSettingsScreen() {
  const { error, data } = useAdminSettingsGet();
  if (!data) {
    return <UnreadyBanner error={error} />;
  }

  const settings = parseAdminSettings(data);

  return <MarketSettingsForm settings={settings} />;
}
