"use client";

import { useMemo } from "react";

import { type Account } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { FeedEmptyState } from "@/components/feed/FeedEmptyState";
import { QuickShare } from "@/components/feed/QuickShare/QuickShare";
import { ThreadReferenceCard } from "@/components/post/ThreadCard";
import { PaginationControls } from "@/components/site/PaginationControls/PaginationControls";
import { Unready } from "@/components/site/Unready";
import { Button } from "@/components/ui/button";
import { CheckIcon } from "@/components/ui/icons/Check";
import { SortAscendingIcon, SortDescendingIcon } from "@/components/ui/icons/Sort";
import * as Menu from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";
import { type Settings } from "@/lib/settings/settings";
import { HStack, LStack, VStack, WStack } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useThreadFeedScreen } from "./useThreadFeedScreen";

export function ThreadFeedScreen({
  initialPage,
  initialPageData,
  category,
  paginationBasePath,
  showCategorySelect,
  hideCategoryBadge = false,
  showQuickShare = true,
  initialSession,
  initialSettings,
}: Props & {
  showCategorySelect: boolean;
  hideCategoryBadge?: boolean;
  showQuickShare?: boolean;
  initialSession?: Account;
  initialSettings?: Settings;
}) {
  const session = useSession(initialSession, initialSettings);

  return (
    <LStack>
      {showQuickShare && (
        <QuickShare
          initialSession={session}
          initialCategory={category}
          showCategorySelect={showCategorySelect}
        />
      )}
      <ThreadFeed
        initialPage={initialPage}
        initialPageData={initialPageData}
        category={category}
        paginationBasePath={paginationBasePath}
        hideCategoryBadge={hideCategoryBadge}
      />
    </LStack>
  );
}

export function ThreadFeed(props: Props & { hideCategoryBadge?: boolean }) {
  const t = useTranslation();
  const {
    ready,
    error,
    showPaginationTop,
    data,
    sortOrder,
    handlePageChange,
    handleSetSortOrder,
  } = useThreadFeedScreen(props);

  const sortedThreads = useMemo(() => {
    if (!data?.threads) return [];
    const threads = [...data.threads];
    threads.sort((a, b) => {
      const timeA = new Date(a.createdAt).getTime();
      const timeB = new Date(b.createdAt).getTime();
      return sortOrder === "asc" ? timeA - timeB : timeB - timeA;
    });
    return threads;
  }, [data?.threads, sortOrder]);

  if (!ready) {
    return <Unready error={error} />;
  }

  if (data.threads.length === 0) {
    return <FeedEmptyState />;
  }

  return (
    <VStack w="full">
      <WStack justifyContent="flex-end" w="full">
        <Menu.Root positioning={{ placement: "bottom-end" }} lazyMount>
          <Menu.Trigger asChild>
            <Button variant="ghost" size="xs" aria-label={t.thread.sortBy}>
              <HStack gap="1" fontSize="xs">
                {sortOrder === "asc" ? (
                  <SortAscendingIcon width="3.5" height="3.5" />
                ) : (
                  <SortDescendingIcon width="3.5" height="3.5" />
                )}
                <span>
                  {sortOrder === "asc"
                    ? t.thread.sortOldestFirst
                    : t.thread.sortNewestFirst}
                </span>
              </HStack>
            </Button>
          </Menu.Trigger>
          <Menu.Positioner>
            <Menu.Content minW="44">
              <Menu.ItemGroup id="feed-sort-options">
                <Menu.Item
                  key="desc"
                  value="desc"
                  onClick={() => handleSetSortOrder("desc")}
                  aria-label={t.thread.sortNewestFirst}
                >
                  <HStack gap="2" justifyContent="space-between" w="full">
                    <HStack gap="2">
                      <SortDescendingIcon width="4" height="4" />
                      <span>{t.thread.sortNewestFirst}</span>
                    </HStack>
                    {sortOrder === "desc" && <CheckIcon width="4" height="4" />}
                  </HStack>
                </Menu.Item>
                <Menu.Item
                  key="asc"
                  value="asc"
                  onClick={() => handleSetSortOrder("asc")}
                  aria-label={t.thread.sortOldestFirst}
                >
                  <HStack gap="2" justifyContent="space-between" w="full">
                    <HStack gap="2">
                      <SortAscendingIcon width="4" height="4" />
                      <span>{t.thread.sortOldestFirst}</span>
                    </HStack>
                    {sortOrder === "asc" && <CheckIcon width="4" height="4" />}
                  </HStack>
                </Menu.Item>
              </Menu.ItemGroup>
            </Menu.Content>
          </Menu.Positioner>
        </Menu.Root>
      </WStack>

      {showPaginationTop && (
        <PaginationControls
          path="/"
          currentPage={data.current_page}
          totalPages={data.total_pages}
          pageSize={data.page_size}
          onClick={handlePageChange}
        />
      )}
      <ol className={lstack()}>
        {sortedThreads.map((t) => {
          return (
            <ThreadReferenceCard
              key={t.slug}
              thread={t}
              hideCategoryBadge={props.hideCategoryBadge}
            />
          );
        })}
      </ol>

      <PaginationControls
        path={props.paginationBasePath}
        currentPage={data.current_page}
        totalPages={data.total_pages}
        pageSize={data.page_size}
        onClick={handlePageChange}
      />
    </VStack>
  );
}
