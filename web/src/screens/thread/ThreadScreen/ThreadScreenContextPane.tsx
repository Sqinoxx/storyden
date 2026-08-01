"use client";

import { formatDate } from "date-fns";

import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { AuthorIcon } from "@/components/ui/icons/Author";
import { CalendarIcon } from "@/components/ui/icons/Calendar";
import { MembersIcon } from "@/components/ui/icons/Members";
import { ParticipatingIcon } from "@/components/ui/icons/Participating";
import { SlugIcon } from "@/components/ui/icons/Slug";
import * as Table from "@/components/ui/table";
import { css, cva } from "@/styled-system/css";
import { Box, HStack, LStack } from "@/styled-system/jsx";
import { Floating } from "@/styled-system/patterns";
import { ScrollToTop } from "@/components/ui/scroll-to-top";
import { useLanguage } from "@/lib/i18n";

import { Props, useThreadScreen } from "./useThreadScreen";

const valueStyles = cva({
  base: {},
  defaultVariants: {
    style: "base",
  },
  variants: {
    style: {
      base: {},
      numeric: {
        fontVariant: "tabular-nums",
        fontFamily: "mono",
      },
    },
  },
});

export function ThreadScreenContextPane(props: Props) {
  const { ready, error, form, isEditing, isEmpty, resetKey, data, handlers } =
    useThreadScreen(props);
  const { t, language } = useLanguage();

  if (!ready) {
    return <Unready error={error} />;
  }

  const { thread } = data;

  const tableData = [
    {
      key: "id",
      label: t.thread?.id ?? "ID",
      icon: SlugIcon,
      value: thread.id,
      style: "numeric" as const,
    },
    {
      key: "author",
      label: t.thread?.author ?? "author",
      icon: AuthorIcon,
      value: (
        <MemberBadge
          profile={thread.author}
          size="xs"
          avatar="hidden"
          name="full-horizontal"
        />
      ),
    },
    {
      key: "started",
      label: t.thread?.started ?? "started",
      icon: CalendarIcon,
      value: formatDate(
        thread.createdAt,
        language === "de" ? "d. MMM yyyy" : "MMM d, yyyy"
      ),
    },
    {
      key: "replies",
      label: t.thread?.replies ?? "replies",
      icon: MembersIcon,
      value: `${thread.reply_status.replies}`,
    },
    {
      key: "participating",
      label: t.thread?.participating ?? "participating",
      icon: ParticipatingIcon,
      value: thread.reply_status.replied
        ? (t.common?.yes ?? "Yes")
        : (t.common?.no ?? "No"),
    },
  ];

  return (
    <Box className={Floating()} borderRadius="md" p="3" w="full">
      <LStack gap="1">
        <Heading>{thread.title}</Heading>
        <p className={css({ color: "fg.muted" })}>{thread.description}</p>

        <Table.Root size="sm" tableLayout="fixed" w="full" overflow="hidden">
          <Table.Body>
            {tableData.map((item) => (
              <Table.Row key={item.key}>
                <Table.Cell fontWeight="medium" color="fg.muted">
                  <HStack gap="1" flexShrink="0">
                    <item.icon width="4" />
                    <span>{item.label}</span>
                  </HStack>
                </Table.Cell>
                <Table.Cell
                  className={valueStyles({ style: item.style })}
                  display="flex"
                  justifyContent="flex-end"
                  alignItems="center"
                  textAlign="right"
                  overflow="hidden"
                  textOverflow="ellipsis"
                  width="full"
                  maxWidth="full"
                  minW="0"
                >
                  {item.value}
                </Table.Cell>
              </Table.Row>
            ))}
          </Table.Body>
        </Table.Root>

        <p>
          <ScrollToTop />
        </p>
      </LStack>
    </Box>
  );
}
