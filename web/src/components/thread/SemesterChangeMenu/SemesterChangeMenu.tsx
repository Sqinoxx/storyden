import { MenuSelectionDetails, Portal } from "@ark-ui/react";
import { useMemo } from "react";

import { handle } from "@/api/client";
import { ThreadReference } from "@/api/openapi-schema";
import { CalendarIcon } from "@/components/ui/icons/Calendar";
import { SubmenuIcon } from "@/components/ui/icons/Submenu";
import * as Menu from "@/components/ui/menu";
import { useFeedMutations } from "@/lib/feed/mutation";
import { useTranslation } from "@/lib/i18n";
import { useSettings } from "@/lib/settings/settings-client";
import { useThreadMutations } from "@/lib/thread/mutation";
import {
  formatSelectableTermLabel,
  parseTermKey,
  selectableTerms,
  termKey,
  threadTerm,
  writeThreadSemesterMeta,
} from "@/lib/thread/semester";
import { HStack } from "@/styled-system/jsx";

type Props = {
  thread: ThreadReference;
};

export function SemesterChangeMenu({ thread }: Props) {
  const t = useTranslation();
  const { updateThread, revalidate: revalidateFeed } = useFeedMutations();
  const { revalidate: revalidateThread } = useThreadMutations(thread);

  const settingsResult = useSettings();
  const legacyLabel = settingsResult.ready
    ? settingsResult.settings.metadata.semester.legacyLabel
    : "";

  const current = termKey(threadTerm(thread));

  const items = useMemo(() => {
    const now = new Date();

    return selectableTerms(now).map((term) => ({
      value: termKey(term),
      label: formatSelectableTermLabel(term, now, legacyLabel),
    }));
  }, [legacyLabel]);

  async function handleSelect({ value }: MenuSelectionDetails) {
    const term = parseTermKey(value);
    if (!term || value === current) return;

    await handle(
      async () => {
        await updateThread(thread.id, {
          meta: writeThreadSemesterMeta(thread.meta, term),
        });
      },
      {
        promiseToast: {
          loading: t.thread.semesterChanging,
          success: t.thread.semesterChanged,
        },
        async cleanup() {
          await revalidateFeed();
          await revalidateThread();
        },
      },
    );
  }

  return (
    <Menu.Root
      size="xs"
      positioning={{ placement: "right-start", gutter: -2 }}
      onSelect={handleSelect}
    >
      <Menu.TriggerItem justifyContent="space-between">
        <HStack gap="1">
          <CalendarIcon />
          {t.thread.changeSemester}
        </HStack>
        <SubmenuIcon />
      </Menu.TriggerItem>

      <Portal>
        <Menu.Positioner>
          <Menu.Content minW="36" maxH="80" overflowY="auto" userSelect="none">
            <Menu.ItemGroup id="semester">
              <Menu.ItemGroupLabel>
                {t.thread.changeSemester}
              </Menu.ItemGroupLabel>

              <Menu.Separator />

              {items.map((item) => (
                <Menu.Item
                  key={item.value}
                  value={item.value}
                  fontWeight={item.value === current ? "bold" : "normal"}
                >
                  {item.label}
                </Menu.Item>
              ))}
            </Menu.ItemGroup>
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}
