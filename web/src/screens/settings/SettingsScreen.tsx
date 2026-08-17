"use client";

import { TabsValueChangeDetails } from "@ark-ui/react";
import { useQueryState } from "nuqs";
import { useEffect } from "react";

import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import * as Tabs from "@/components/ui/tabs";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { Settings } from "@/lib/settings/settings";
import { hasPermission } from "@/utils/permissions";

import { MemberAccessKeysSettingsScreen } from "./MemberAccessKeysSettingsScreen";
import { MemberAccountSettingsScreen } from "./MemberAccountSettingsScreen";
import { MemberAuthenticationSettingsScreen } from "./MemberAuthenticationSettingsScreen";
import { MemberEmailSettingsScreen } from "./MemberEmailSettingsScreen";
import { MemberInterfaceSettingsScreen } from "./MemberInterfaceSettingsScreen";
import { MemberInvitationsSettingsScreen } from "./MemberInvitationsSettingsScreen";
import { MemberOAuthSettingsScreen } from "./MemberOAuthSettingsScreen";
import { MemberSessionSettingsScreen } from "./MemberSessionSettingsScreen";

const DEFAULT_TAB = "interface";

type Props = {
  initialSettings: Settings;
};

export function SettingsScreen({ initialSettings }: Props) {
  const { t } = useLanguage();
  const session = useSession();
  const [tab, setTab] = useQueryState("tab", {
    defaultValue: DEFAULT_TAB,
  });

  // NOTE: A hack because for some reason, the tab component renders twice and
  // the associated hook gets lost and results in `ready` always being false,
  // despite the useSettings hook returning the correct data. Not sure if this
  // is a Next.js bug, a React bug or a Ark, Park or something else bug...
  useEffect(() => {
    if (!tab) {
      setTab(DEFAULT_TAB);
    }
  }, [tab, setTab]);

  function handleTabChange({ value }: TabsValueChangeDetails) {
    setTab(value);
  }

  const emailEnabled = initialSettings.capabilities.includes("email_client");
  const oauthCapabilityEnabled = initialSettings.capabilities.includes("oauth");

  const accessKeysEnabled = hasPermission(
    session,
    Permission.USE_PERSONAL_ACCESS_KEYS,
  );
  const oauthEnabled =
    oauthCapabilityEnabled && hasPermission(session, Permission.ADMINISTRATOR);
  const invitationsEnabled = hasPermission(
    session,
    Permission.CREATE_INVITATION,
  );

  const availableTabs = [
    "interface",
    "authentication",
    ...(emailEnabled ? ["email"] : []),
    ...(invitationsEnabled ? ["invitations"] : []),
    ...(accessKeysEnabled ? ["access_keys"] : []),
    "session",
    "account",
    ...(oauthEnabled ? ["oauth"] : []),
  ];
  const activeTab = tab && availableTabs.includes(tab) ? tab : DEFAULT_TAB;

  return (
    <Tabs.Root
      key={availableTabs.join(",")}
      width="full"
      variant="enclosed"
      defaultValue={DEFAULT_TAB}
      value={activeTab}
      onValueChange={handleTabChange}
    >
      <Tabs.List>
        <Tabs.Trigger value="interface">
          {t.settings?.tabs?.interface || "Interface"}
        </Tabs.Trigger>
        <Tabs.Trigger value="authentication">
          {t.settings?.tabs?.authentication || "Authentication"}
        </Tabs.Trigger>
        {emailEnabled && (
          <Tabs.Trigger value="email">
            {t.settings?.tabs?.email || "Email"}
          </Tabs.Trigger>
        )}
        {invitationsEnabled && (
          <Tabs.Trigger value="invitations">
            {t.settings?.tabs?.invitations || "Invitations"}
          </Tabs.Trigger>
        )}
        {accessKeysEnabled && (
          <Tabs.Trigger value="access_keys">
            {t.settings?.tabs?.accessKeys || "Access keys"}
          </Tabs.Trigger>
        )}
        <Tabs.Trigger value="session">
          {t.settings?.tabs?.session || "Session"}
        </Tabs.Trigger>
        <Tabs.Trigger value="account">
          {t.settings?.tabs?.account || "Konto"}
        </Tabs.Trigger>
        {oauthEnabled && (
          <Tabs.Trigger value="oauth">
            {t.settings?.tabs?.oauth || "OAuth"}
          </Tabs.Trigger>
        )}
        <Tabs.Indicator />
      </Tabs.List>

      <Tabs.Content value="interface">
        <MemberInterfaceSettingsScreen />
      </Tabs.Content>

      <Tabs.Content value="authentication">
        <MemberAuthenticationSettingsScreen />
      </Tabs.Content>

      {emailEnabled && (
        <Tabs.Content value="email">
          <MemberEmailSettingsScreen />
        </Tabs.Content>
      )}

      {invitationsEnabled && (
        <Tabs.Content value="invitations">
          <MemberInvitationsSettingsScreen />
        </Tabs.Content>
      )}

      {accessKeysEnabled && (
        <Tabs.Content value="access_keys">
          <MemberAccessKeysSettingsScreen />
        </Tabs.Content>
      )}

      <Tabs.Content value="session">
        <MemberSessionSettingsScreen />
      </Tabs.Content>

      <Tabs.Content value="account">
        <MemberAccountSettingsScreen />
      </Tabs.Content>

      {oauthEnabled && (
        <Tabs.Content value="oauth">
          <MemberOAuthSettingsScreen />
        </Tabs.Content>
      )}
    </Tabs.Root>
  );
}
