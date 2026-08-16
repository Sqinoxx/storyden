"use client";

import { useAdminSettingsGet } from "@/api/openapi-client/admin";
import { UnreadyBanner } from "@/components/site/Unready";
import { parseAdminSettings } from "@/lib/settings/settings";

import { SessionSettingsForm } from "../../components/admin/SessionSettings/SessionSettings";

export function SessionSettingsScreen() {
  const { error, data } = useAdminSettingsGet();
  if (!data) {
    return <UnreadyBanner error={error} />;
  }

  const settings = parseAdminSettings(data);

  return <SessionSettingsForm settings={settings} />;
}
