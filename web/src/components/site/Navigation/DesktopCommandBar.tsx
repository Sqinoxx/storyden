import { getServerSession } from "@/auth/server-session";
import { parseMemberSettings } from "@/lib/settings/member-settings";
import { allowsPublicRegistration } from "@/lib/settings/registration";
import { getSettings } from "@/lib/settings/settings-server";
import { cx } from "@/styled-system/css";
import { HStack } from "@/styled-system/jsx";
import { Floating } from "@/styled-system/patterns";

import styles from "./navigation.module.css";

import { SearchAnchor } from "./Anchors/Search";
import { MemberActions } from "./MemberActions";
import { SidebarToggle } from "./NavigationPane/SidebarToggle";
import { getServerSidebarState } from "./NavigationPane/server";
import { ThemeToggle } from "./ThemeToggle";
import { Title } from "./Title";

export async function DesktopCommandBar() {
  const globalSettings = await getSettings();
  const { title, registration_mode } = globalSettings;
  const sessionAccount = await getServerSession();
  const session = sessionAccount
    ? parseMemberSettings(sessionAccount, globalSettings.metadata)
    : undefined;

  const sidebarDefaultState =
    session?.meta.sidebar.defaultState ??
    globalSettings.metadata.sidebar.defaultState;
  const initialSidebarState = await getServerSidebarState(sidebarDefaultState);

  const canRegister = allowsPublicRegistration(registration_mode);

  return (
    <HStack
      className={cx(Floating(), styles["topbar"])}
      borderRadius="md"
      justify="space-between"
      alignItems="center"
      px="1"
    >
      <HStack className={styles["topbar-left"]}>
        {sessionAccount && <SidebarToggle initialValue={initialSidebarState} />}
        {sessionAccount && <SearchAnchor />}
      </HStack>

      <HStack className={styles["topbar-middle"]} justify="space-around">
        <Title>StuDent</Title>
      </HStack>

      <HStack className={styles["topbar-right"]}>
        <ThemeToggle />
        <MemberActions session={sessionAccount} canRegister={canRegister} />
      </HStack>
    </HStack>
  );
}
