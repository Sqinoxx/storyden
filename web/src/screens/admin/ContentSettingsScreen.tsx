"use client";

import { useAdminSettingsGet } from "@/api/openapi-client/admin";
import { UnreadyBanner } from "@/components/site/Unready";
import { parseAdminSettings } from "@/lib/settings/settings";

import { ContentSettingsForm } from "../../components/admin/ContentSettings/ContentSettings";

export function ContentSettingsScreen() {
  const { error, data } = useAdminSettingsGet();
  if (!data) {
    return <UnreadyBanner error={error} />;
  }

  const settings = parseAdminSettings(data);

  return <ContentSettingsForm settings={settings} />;
}
