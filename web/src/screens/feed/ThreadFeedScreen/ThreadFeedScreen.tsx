"use client";

import { type Account } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { FeedEmptyState } from "@/components/feed/FeedEmptyState";
import { QuickShare } from "@/components/feed/QuickShare/QuickShare";
import { SemesterGroupedThreadList } from "@/components/feed/SemesterGroups/SemesterGroupedThreadList";
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

import {
  Props,
  SEMESTER_VIEW,
  ThreadFeedView,
  useThreadFeedScreen,
} from "./useThreadFeedScreen";

const SORT_ORDERS: ThreadFeedView[] = ["newest", "activity", "oldest"];

function viewLabels(
  t: ReturnType<typeof useTranslation>,
): Record<ThreadFeedView, string> {
  return {
    newest: t.thread.sortNewestFirst,
    activity: t.thread.sortRecentActivity,
    oldest: t.thread.sortOldestFirst,
    [SEMESTER_VIEW]: t.thread.sortBySemester,
  };
}

export function ThreadFeedScreen({
  initialPage,
  initialPageData,
  category,
  paginationBasePath,
  showCategorySelect,
  hideCategoryBadge = false,
  showQuickShare = true,
  enableSemesterGrouping = false,
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
        enableSemesterGrouping={enableSemesterGrouping}
      />
    </LStack>
  );
}

export function ThreadFeed(props: Props & { hideCategoryBadge?: boolean }) {
  const t = useTranslation();
  const {
    ready,
    error,
    data,
    view,
    isGrouped,
    handlePageChange,
    handleSetView,
  } = useThreadFeedScreen(props);

  if (!ready) {
    return <Unready error={error} />;
  }

  if (data.threads.length === 0) {
    return <FeedEmptyState />;
  }

  const views: ThreadFeedView[] = props.enableSemesterGrouping
    ? [...SORT_ORDERS, SEMESTER_VIEW]
    : SORT_ORDERS;

  return (
    <VStack w="full">
      <WStack justifyContent="flex-end" w="full">
        <Menu.Root positioning={{ placement: "bottom-end" }} lazyMount>
          <Menu.Trigger asChild>
            <Button variant="ghost" size="xs" aria-label={t.thread.sortBy}>
              <HStack gap="1" fontSize="xs">
                {view === "oldest" ? (
                  <SortAscendingIcon width="3.5" height="3.5" />
                ) : (
                  <SortDescendingIcon width="3.5" height="3.5" />
                )}
                <span>{viewLabels(t)[view]}</span>
              </HStack>
            </Button>
          </Menu.Trigger>
          <Menu.Positioner>
            <Menu.Content minW="44">
              <Menu.ItemGroup id="feed-sort-options">
                {views.map((option) => {
                  const label = viewLabels(t)[option];
                  const Icon =
                    option === "oldest" ? SortAscendingIcon : SortDescendingIcon;

                  return (
                    <Menu.Item
                      key={option}
                      value={option}
                      onClick={() => handleSetView(option)}
                      aria-label={label}
                      display={
                        option === SEMESTER_VIEW
                          ? { base: "none", md: "flex" }
                          : undefined
                      }
                    >
                      <HStack gap="2" justifyContent="space-between" w="full">
                        <HStack gap="2">
                          <Icon width="4" height="4" />
                          <span>{label}</span>
                        </HStack>
                        {view === option && <CheckIcon width="4" height="4" />}
                      </HStack>
                    </Menu.Item>
                  );
                })}
              </Menu.ItemGroup>
            </Menu.Content>
          </Menu.Positioner>
        </Menu.Root>
      </WStack>

      <PaginationControls
        path={props.paginationBasePath}
        currentPage={data.current_page}
        totalPages={data.total_pages}
        pageSize={data.page_size}
        onClick={handlePageChange}
      />
      {isGrouped ? (
        <SemesterGroupedThreadList
          threads={data.threads}
          hideCategoryBadge={props.hideCategoryBadge}
        />
      ) : (
        <ol className={lstack()}>
          {data.threads.map((t) => {
            return (
              <ThreadReferenceCard
                key={t.slug}
                thread={t}
                hideCategoryBadge={props.hideCategoryBadge}
              />
            );
          })}
        </ol>
      )}

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
