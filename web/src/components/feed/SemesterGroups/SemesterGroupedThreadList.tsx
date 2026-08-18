"use client";

import { ThreadReference } from "@/api/openapi-schema";
import { ThreadReferenceCard } from "@/components/post/ThreadCard";
import { useTranslation } from "@/lib/i18n";
import { formatTermLabel, groupThreadsBySemester } from "@/lib/thread/semester";
import { HStack, LStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

type Props = {
  threads: ThreadReference[];
  hideCategoryBadge?: boolean;
};

function SemesterRail({ label }: { label: string }) {
  return (
    <HStack
      display={{ base: "none", md: "flex" }}
      gap="1"
      alignItems="stretch"
      aria-hidden="true"
    >
      <styled.div
        w="2"
        borderStyle="solid"
        borderColor="border.subtle"
        borderRightWidth="thin"
        borderTopWidth="thin"
        borderBottomWidth="thin"
        borderTopRightRadius="sm"
        borderBottomRightRadius="sm"
      />
      <styled.span
        fontSize="xs"
        color="fg.subtle"
        letterSpacing="wide"
        style={{ writingMode: "vertical-rl" }}
      >
        {label}
      </styled.span>
    </HStack>
  );
}

export function SemesterGroupedThreadList({
  threads,
  hideCategoryBadge,
}: Props) {
  const t = useTranslation();
  const groups = groupThreadsBySemester(threads);

  return (
    <LStack gap="4">
      {groups.map((group) => {
        const label = group.term
          ? formatTermLabel(group.term)
          : t.thread.pinnedGroup;

        return (
          <styled.section
            key={group.key}
            w="full"
            display="grid"
            gridTemplateColumns="1fr auto"
            columnGap="2"
            alignItems="stretch"
            aria-label={label}
          >
            <HStack gridColumn="1 / -1" gap="2" w="full" pb="2">
              <styled.span
                fontSize="xs"
                fontWeight="medium"
                color="fg.muted"
                flexShrink="0"
              >
                {label}
              </styled.span>
              <styled.span
                flexGrow="1"
                h="0"
                borderTopWidth="thin"
                borderTopStyle="solid"
                borderColor="border.subtle"
              />
            </HStack>

            <ol className={lstack()}>
              {group.threads.map((thread) => (
                <ThreadReferenceCard
                  key={thread.slug}
                  thread={thread}
                  hideCategoryBadge={hideCategoryBadge}
                />
              ))}
            </ol>

            {group.term && <SemesterRail label={label} />}
          </styled.section>
        );
      })}
    </LStack>
  );
}
