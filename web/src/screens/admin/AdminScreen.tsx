"use client";

import { TabsValueChangeDetails } from "@ark-ui/react";
import { useQueryState } from "nuqs";
import { useEffect } from "react";

import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import * as Tabs from "@/components/ui/tabs";
import { useCapability } from "@/lib/settings/capabilities";
import { hasPermission } from "@/utils/permissions";

import { AccessKeySettingsScreen } from "./AccessKeySettingsScreen";
import { AuditLogSettingsScreen } from "./AuditLogSettingsScreen/AuditLogSettingsScreen";
import { AuthenticationSettingsScreen } from "./AuthenticationSettingsScreen";
import { BrandSettingsScreen } from "./BrandSettingsScreen";
import { ContentSettingsScreen } from "./ContentSettingsScreen";
import { DriveSettingsScreen } from "./DriveSettingsScreen";
import { EmailLogSettingsScreen } from "./EmailLogSettingsScreen/EmailLogSettingsScreen";
import { InterfaceSettingsScreen } from "./InterfaceSettingsScreen";
import { ModerationSettingsScreen } from "./ModerationSettingsScreen";
import { OAuthSettingsScreen } from "./OAuthSettingsScreen";
import { OCRSettingsScreen } from "./OCRSettingsScreen/OCRSettingsScreen";
import { PluginSettingsScreen } from "./PluginSettingsScreen";
import { RobotsSettingsScreen } from "./RobotsSettingsScreen/RobotsSettingsScreen";
import { SessionSettingsScreen } from "./SessionSettingsScreen";
import { StatisticsSettingsScreen } from "./StatisticsSettingsScreen/StatisticsSettingsScreen";
import { SystemSettingsScreen } from "./SystemSettingsScreen";

const DEFAULT_TAB = "brand";

export function AdminScreen() {
  const pluginsEnabled = useCapability("plugins");
  const oauthCapabilityEnabled = useCapability("oauth");
  const session = useSession();
  const canViewEmailLog = hasPermission(session, Permission.ADMINISTRATOR);
  const canViewStatistics = hasPermission(session, Permission.ADMINISTRATOR);
  const canViewOAuth =
    oauthCapabilityEnabled && hasPermission(session, Permission.ADMINISTRATOR);
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
      return;
    }

    if (!pluginsEnabled && tab === "plugins") {
      setTab(DEFAULT_TAB);
    }

    if (!canViewEmailLog && tab === "email") {
      setTab(DEFAULT_TAB);
    }

    if (!canViewStatistics && tab === "statistics") {
      setTab(DEFAULT_TAB);
    }

    if (!canViewOAuth && tab === "oauth") {
      setTab(DEFAULT_TAB);
    }
  }, [
    canViewEmailLog,
    canViewStatistics,
    canViewOAuth,
    pluginsEnabled,
    tab,
    setTab,
  ]);

  function handleTabChange({ value }: TabsValueChangeDetails) {
    setTab(value);
  }

  return (
    <Tabs.Root
      width="full"
      variant="enclosed"
      // variant="line"
      // variant="outline"
      defaultValue={DEFAULT_TAB}
      value={tab}
      onValueChange={handleTabChange}
    >
      <Tabs.List
        flexWrap="wrap"
        overflow="visible"
        height="auto!"
        rowGap="1"
        columnGap="1"
        py="1"
      >
        <Tabs.Trigger value="brand">Brand</Tabs.Trigger>
        <Tabs.Trigger value="content">Content</Tabs.Trigger>
        <Tabs.Trigger value="moderation">Moderation</Tabs.Trigger>
        <Tabs.Trigger value="system">System</Tabs.Trigger>
        <Tabs.Trigger value="ocr">OCR & Bildtext</Tabs.Trigger>
        {canViewStatistics && (
          <Tabs.Trigger value="statistics">Statistiken</Tabs.Trigger>
        )}
        <Tabs.Trigger value="audit">Audit Log</Tabs.Trigger>
        {canViewEmailLog && <Tabs.Trigger value="email">Email</Tabs.Trigger>}
        <Tabs.Trigger value="interface">Interface</Tabs.Trigger>
        <Tabs.Trigger value="authentication">Authentication</Tabs.Trigger>
        <Tabs.Trigger value="sessions">Sessions</Tabs.Trigger>
        <Tabs.Trigger value="access_keys">Access keys</Tabs.Trigger>
        {canViewOAuth && <Tabs.Trigger value="oauth">OAuth</Tabs.Trigger>}
        {pluginsEnabled && <Tabs.Trigger value="plugins">Plugins</Tabs.Trigger>}
        <Tabs.Trigger value="drive">Drive</Tabs.Trigger>
        <Tabs.Trigger value="robots">Robots</Tabs.Trigger>
        <Tabs.Indicator />
      </Tabs.List>

      <Tabs.Content value="brand">
        {tab === "brand" && <BrandSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="content">
        {tab === "content" && <ContentSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="moderation">
        {tab === "moderation" && <ModerationSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="system">
        {tab === "system" && <SystemSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="ocr">
        {tab === "ocr" && <OCRSettingsScreen />}
      </Tabs.Content>

      {canViewStatistics && (
        <Tabs.Content value="statistics">
          {tab === "statistics" && <StatisticsSettingsScreen />}
        </Tabs.Content>
      )}

      <Tabs.Content value="audit">
        {tab === "audit" && <AuditLogSettingsScreen />}
      </Tabs.Content>

      {canViewEmailLog && (
        <Tabs.Content value="email">
          {tab === "email" && <EmailLogSettingsScreen />}
        </Tabs.Content>
      )}

      <Tabs.Content value="interface">
        {tab === "interface" && <InterfaceSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="authentication">
        {tab === "authentication" && <AuthenticationSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="sessions">
        {tab === "sessions" && <SessionSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="access_keys">
        {tab === "access_keys" && <AccessKeySettingsScreen />}
      </Tabs.Content>

      {canViewOAuth && (
        <Tabs.Content value="oauth">
          {tab === "oauth" && <OAuthSettingsScreen />}
        </Tabs.Content>
      )}

      {pluginsEnabled && (
        <Tabs.Content value="plugins">
          {tab === "plugins" && <PluginSettingsScreen />}
        </Tabs.Content>
      )}

      <Tabs.Content value="drive">
        {tab === "drive" && <DriveSettingsScreen />}
      </Tabs.Content>

      <Tabs.Content value="robots">
        {tab === "robots" && <RobotsSettingsScreen />}
      </Tabs.Content>
    </Tabs.Root>
  );
}
