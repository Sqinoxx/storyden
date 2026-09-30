"use client";

import { useMemo, useState } from "react";
import useSWR from "swr";

import { handle } from "@/api/client";
import {
  nodeListChildren,
  nodeUpdateVisibility,
  useNodeGet,
} from "@/api/openapi-client/nodes";
import { NodeWithChildren, Visibility } from "@/api/openapi-schema";
import { UnreadyBanner } from "@/components/site/Unready";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Input } from "@/components/ui/input";
import { LinkButton } from "@/components/ui/link-button";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { Box, Flex, HStack, LStack, styled } from "@/styled-system/jsx";

const ARCHIVE_SLUG = "tagesarchiv";
const ROUTE = "/admin/tagesarchiv";

type Filter = "pending" | "published" | "all";

type Day = {
  slug: string;
  name: string;
  date: string;
  threadCount: number;
  published: boolean;
};

async function fetchAllDays(): Promise<Day[]> {
  const nodes: NodeWithChildren[] = [];
  let page: number | undefined = 1;

  while (page) {
    const result = await nodeListChildren(ARCHIVE_SLUG, {
      page: String(page),
    });
    nodes.push(...result.nodes);
    page = result.next_page;
  }

  return nodes
    .map((node) => {
      const meta = (node.meta?.["daily_library"] ?? {}) as {
        date?: string;
        thread_count?: number;
      };

      return {
        slug: node.slug,
        name: node.name,
        date: meta.date ?? node.slug.replace(`${ARCHIVE_SLUG}-`, ""),
        threadCount: meta.thread_count ?? 0,
        published: node.visibility === Visibility.published,
      };
    })
    .sort((a, b) => b.date.localeCompare(a.date));
}

export function DailyLibraryScreen() {
  const t = useTranslation();

  const archive = useNodeGet(ARCHIVE_SLUG);
  const days = useSWR(
    archive.data ? ["daily-library-days", ARCHIVE_SLUG] : null,
    fetchAllDays,
  );

  const [filter, setFilter] = useState<Filter>("pending");
  const [search, setSearch] = useState("");
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [busy, setBusy] = useState(false);

  const all = days.data ?? [];
  const pendingCount = all.filter((d) => !d.published).length;

  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    return all.filter((d) => {
      if (filter === "pending" && d.published) return false;
      if (filter === "published" && !d.published) return false;
      if (!q) return true;
      return d.name.toLowerCase().includes(q) || d.date.includes(q);
    });
  }, [all, filter, search]);

  const selectedVisible = visible.filter((d) => selected.has(d.slug));
  const allVisibleSelected =
    visible.length > 0 && selectedVisible.length === visible.length;

  function toggle(slug: string, checked: boolean) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (checked) next.add(slug);
      else next.delete(slug);
      return next;
    });
  }

  function toggleAll(checked: boolean) {
    setSelected(checked ? new Set(visible.map((d) => d.slug)) : new Set());
  }

  async function setVisibility(slugs: string[], visibility: Visibility) {
    setBusy(true);
    await handle(
      async () => {
        for (const slug of slugs) {
          await nodeUpdateVisibility(slug, { visibility });
        }
      },
      {
        errorToast: true,
        cleanup: async () => {
          setBusy(false);
          setSelected(new Set());
          await Promise.all([days.mutate(), archive.mutate()]);
        },
      },
    );
  }

  if (archive.error) {
    const status = (archive.error as { status?: number }).status;
    if (status === 404) {
      return (
        <LStack gap="6" width="full">
          <Header />
          <EmptyBox>{t.dailyLibrary.notGenerated}</EmptyBox>
        </LStack>
      );
    }
    return <UnreadyBanner error={archive.error} />;
  }

  if (!archive.data) {
    return <UnreadyBanner />;
  }

  const archivePublished = archive.data.visibility === Visibility.published;

  return (
    <LStack gap="6" width="full">
      <Header />

      {!archivePublished && (
        <Flex
          p="4"
          gap="4"
          width="full"
          alignItems="center"
          justifyContent="space-between"
          flexWrap="wrap"
          borderRadius="l2"
          borderWidth="thin"
          borderColor="border.default"
          bg="bg.subtle"
        >
          <Text textStyle="sm">{t.dailyLibrary.archiveUnpublished}</Text>
          <Button
            size="sm"
            variant="solid"
            disabled={busy}
            onClick={() => setVisibility([ARCHIVE_SLUG], Visibility.published)}
          >
            {t.dailyLibrary.publishArchive}
          </Button>
        </Flex>
      )}

      <HStack gap="4" width="full">
        <Stat label={t.dailyLibrary.totalDays} value={all.length} />
        <Stat label={t.dailyLibrary.pendingDays} value={pendingCount} />
        <Stat
          label={t.dailyLibrary.publishedDays}
          value={all.length - pendingCount}
        />
      </HStack>

      <Flex gap="3" width="full" flexWrap="wrap" alignItems="center">
        <HStack gap="1">
          {(
            [
              ["pending", t.dailyLibrary.filterPending],
              ["published", t.dailyLibrary.filterPublished],
              ["all", t.dailyLibrary.filterAll],
            ] as const
          ).map(([value, label]) => (
            <Button
              key={value}
              size="sm"
              variant={filter === value ? "solid" : "outline"}
              onClick={() => {
                setFilter(value);
                setSelected(new Set());
              }}
            >
              {label}
            </Button>
          ))}
        </HStack>
        <Input
          size="sm"
          flex="1"
          minW="48"
          placeholder={t.dailyLibrary.searchPlaceholder}
          value={search}
          onChange={(e) => setSearch(e.target.value)}
        />
      </Flex>

      <Flex
        width="full"
        gap="3"
        alignItems="center"
        justifyContent="space-between"
        flexWrap="wrap"
      >
        <Checkbox
          size="sm"
          checked={allVisibleSelected}
          disabled={visible.length === 0}
          onCheckedChange={({ checked }) => toggleAll(checked === true)}
        >
          {t.dailyLibrary.selectAll}
        </Checkbox>
        <Button
          size="sm"
          variant="solid"
          loading={busy}
          disabled={
            busy || selectedVisible.filter((d) => !d.published).length === 0
          }
          onClick={() =>
            setVisibility(
              selectedVisible.filter((d) => !d.published).map((d) => d.slug),
              Visibility.published,
            )
          }
        >
          {t.dailyLibrary.publishSelected.replace(
            "{count}",
            String(selectedVisible.filter((d) => !d.published).length),
          )}
        </Button>
      </Flex>

      {days.error ? (
        <EmptyBox>{t.dailyLibrary.loadError}</EmptyBox>
      ) : visible.length === 0 ? (
        <EmptyBox>{days.data ? t.dailyLibrary.emptyState : "…"}</EmptyBox>
      ) : (
        <LStack gap="2" width="full">
          {visible.map((day) => (
            <Flex
              key={day.slug}
              width="full"
              p="3"
              gap="3"
              alignItems="center"
              justifyContent="space-between"
              flexWrap="wrap"
              borderRadius="l2"
              borderWidth="thin"
              borderColor="border.subtle"
              bg="bg.default"
            >
              <HStack gap="3">
                <Checkbox
                  size="sm"
                  checked={selected.has(day.slug)}
                  onCheckedChange={({ checked }) =>
                    toggle(day.slug, checked === true)
                  }
                  aria-label={day.name}
                />
                <LStack gap="0">
                  <Text fontWeight="semibold">{day.name}</Text>
                  <Text textStyle="xs" color="fg.muted">
                    {t.dailyLibrary.threadCount.replace(
                      "{count}",
                      String(day.threadCount),
                    )}
                  </Text>
                </LStack>
              </HStack>

              <HStack gap="2">
                <Badge size="sm" variant={day.published ? "solid" : "outline"}>
                  {day.published
                    ? t.dailyLibrary.statusPublished
                    : t.dailyLibrary.statusPending}
                </Badge>
                <LinkButton size="sm" variant="ghost" href={`/l/${day.slug}`}>
                  {t.dailyLibrary.view}
                </LinkButton>
                {day.published ? (
                  <Button
                    size="sm"
                    variant="outline"
                    disabled={busy}
                    onClick={() => setVisibility([day.slug], Visibility.review)}
                  >
                    {t.dailyLibrary.unpublish}
                  </Button>
                ) : (
                  <Button
                    size="sm"
                    variant="solid"
                    disabled={busy}
                    onClick={() =>
                      setVisibility([day.slug], Visibility.published)
                    }
                  >
                    {t.dailyLibrary.publish}
                  </Button>
                )}
              </HStack>
            </Flex>
          ))}
        </LStack>
      )}
    </LStack>
  );
}

function Header() {
  const t = useTranslation();
  return (
    <LStack gap="2">
      <Breadcrumbs
        index={{ label: t.nav.admin, href: "/admin" }}
        crumbs={[{ label: t.dailyLibrary.title, href: ROUTE }]}
      />
      <styled.h1 textStyle="2xl" fontWeight="bold">
        {t.dailyLibrary.title}
      </styled.h1>
      <Text textStyle="sm" color="fg.muted">
        {t.dailyLibrary.subtitle}
      </Text>
    </LStack>
  );
}

function Stat({ label, value }: { label: string; value: number }) {
  return (
    <Box
      flex="1"
      p="4"
      borderRadius="l2"
      borderWidth="thin"
      borderColor="border.subtle"
      bg="bg.default"
    >
      <Text textStyle="xs" color="fg.muted" fontWeight="medium">
        {label}
      </Text>
      <Text textStyle="2xl" fontWeight="bold" mt="1">
        {value}
      </Text>
    </Box>
  );
}

function EmptyBox({ children }: { children: React.ReactNode }) {
  return (
    <Box
      width="full"
      py="12"
      px="4"
      textAlign="center"
      borderWidth="thin"
      borderStyle="dashed"
      borderColor="border.subtle"
      borderRadius="l2"
    >
      <Text color="fg.muted">{children}</Text>
    </Box>
  );
}
