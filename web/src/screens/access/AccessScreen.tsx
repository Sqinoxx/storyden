"use client";

import { Lock } from "lucide-react";
import { useQueryState } from "nuqs";

import { useAccessPassList } from "@/api/openapi-client/access";
import {
  AccessPass,
  AccessPassListResult,
  Permission,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { AccessPassCard } from "@/components/access/AccessPassCard";
import { GrantsTab } from "@/components/access/manage/GrantsTab";
import { PassesTab } from "@/components/access/manage/PassesTab";
import {
  RequestsTab,
  useOpenRequests,
} from "@/components/access/manage/RequestsTab";
import { EmptyState } from "@/components/site/EmptyState";
import { Unready } from "@/components/site/Unready";
import { Badge } from "@/components/ui/badge";
import { Heading } from "@/components/ui/heading";
import * as Tabs from "@/components/ui/tabs";
import { useIncludedNames } from "@/lib/access/useIncludedNames";
import { useTranslation } from "@/lib/i18n";
import { LStack, styled } from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";

type Props = {
  initialPasses: AccessPassListResult;
};

const TABS = ["mine", "requests", "grants", "passes"] as const;

export function AccessScreen({ initialPasses }: Props) {
  const t = useTranslation();
  const session = useSession();
  const canManage = hasPermission(session, Permission.MANAGE_ACCESS);

  const { data, error } = useAccessPassList({
    swr: { fallbackData: initialPasses },
  });

  if (!data) return <Unready error={error} />;

  return (
    <LStack gap="4" w="full">
      <Heading size="lg">{t.access.title}</Heading>
      <styled.p color="fg.muted">{t.access.intro}</styled.p>

      {canManage ? (
        <ManagerTabs passes={data.passes} />
      ) : (
        <MyPasses passes={data.passes} />
      )}
    </LStack>
  );
}

function ManagerTabs({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const m = t.access.manage;
  const { data: requests } = useOpenRequests();
  const [tab, setTab] = useQueryState("tab", { defaultValue: "requests" });
  const current = (TABS as readonly string[]).includes(tab) ? tab : "requests";
  const openCount = requests?.grants.length ?? 0;

  return (
    <Tabs.Root
      width="full"
      variant="enclosed"
      value={current}
      onValueChange={({ value }) => setTab(value)}
      lazyMount
    >
      <Tabs.List flexWrap="wrap" height="auto!" rowGap="1" columnGap="1" py="1">
        <Tabs.Trigger value="requests">
          {m.tabRequests}
          {openCount > 0 && (
            <Badge size="sm" variant="solid" ml="1">
              {openCount}
            </Badge>
          )}
        </Tabs.Trigger>
        <Tabs.Trigger value="grants">{m.tabGrants}</Tabs.Trigger>
        <Tabs.Trigger value="passes">{m.tabPasses}</Tabs.Trigger>
        <Tabs.Trigger value="mine">{m.tabMine}</Tabs.Trigger>
        <Tabs.Indicator />
      </Tabs.List>
      <Tabs.Content value="requests" pt="3">
        <RequestsTab passes={passes} />
      </Tabs.Content>
      <Tabs.Content value="grants" pt="3">
        <GrantsTab passes={passes} />
      </Tabs.Content>
      <Tabs.Content value="passes" pt="3">
        <PassesTab passes={passes} />
      </Tabs.Content>
      <Tabs.Content value="mine" pt="3">
        <MyPasses passes={passes} />
      </Tabs.Content>
    </Tabs.Root>
  );
}

function MyPasses({ passes }: { passes: AccessPass[] }) {
  const t = useTranslation();
  const includedNames = useIncludedNames();

  if (passes.length === 0) {
    return (
      <EmptyState
        icon={<Lock />}
        hideContributionLabel
        w="full"
        unauthenticatedLabel=""
      >
        <p>{t.access.noPasses}</p>
      </EmptyState>
    );
  }

  return (
    <LStack gap="3" w="full">
      {passes.map((p) => (
        <AccessPassCard key={p.id} pass={p} categoryNames={includedNames(p)} />
      ))}
    </LStack>
  );
}
