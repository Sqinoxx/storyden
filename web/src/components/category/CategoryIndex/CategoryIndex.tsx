"use client";

import Image from "next/image";

import { Account, ThreadListResult } from "@/api/openapi-schema";
import { ComposeAnchor } from "@/components/site/Navigation/Anchors/Compose";
import { Heading } from "@/components/ui/heading";
import { CategoryTree } from "@/lib/category/tree";
import { Settings } from "@/lib/settings/settings";
import { ThreadFeedScreen } from "@/screens/feed/ThreadFeedScreen/ThreadFeedScreen";
import { css } from "@/styled-system/css";
import { HStack, LStack, WStack } from "@/styled-system/jsx";

import { CategoryBadge } from "../CategoryBadge";
import { CategoryCreateTrigger } from "../CategoryCreate/CategoryCreateTrigger";

import { CategoryLayout } from "./CategoryCardLayout";
import { useTranslation } from "@/lib/i18n";
import { useSession } from "@/auth";
import { Permission } from "@/api/openapi-schema";
import { hasPermission } from "@/utils/permissions";

export type Props = {
  initialSession?: Account;
  initialSettings?: Settings;
  initialThreadList?: ThreadListResult;
  initialThreadListPage?: number;

  layout: "grid" | "list";
  threadListMode: "none" | "all" | "uncategorised";
  showQuickShare: boolean;
  categories: CategoryTree[];
  paginationBasePath: string;
};

export function CategoryIndex({
  initialSession,
  initialSettings,
  initialThreadList,
  initialThreadListPage,
  layout,
  threadListMode,
  showQuickShare,
  categories,
  paginationBasePath,
}: Props) {
  const t = useTranslation();
  const session = useSession(initialSession, initialSettings);

  // Gate the QuickShare box: only show it when the feature is enabled AND the
  // user has ManageCategories (admin/mod). Normal users cannot post without a
  // leaf category, and this index view only lists top-level / uncategorised
  // threads, so there's nothing valid for them to post into here.
  const canPost = hasPermission(session, Permission.MANAGE_CATEGORIES);
  const effectiveShowQuickShare = showQuickShare && canPost;

  return (
    <LStack gap="8">
      <LStack>
        <WStack alignItems="center">
          <Image
            src="/StuDent-transparent.png"
            alt={t.category.discussionCategories}
            width={1200}
            height={510}
            className={css({
              h: { base: "16", md: "12" },
              w: "auto",
            })}
          />

          <CategoryCreateTrigger />
        </WStack>

        <HStack>
          {categories.map((c) => (
            <CategoryBadge key={c.id} category={c} />
          ))}
        </HStack>

        <CategoryLayout layout={layout} categories={categories} />
      </LStack>

      <ThreadListSection
        initialThreadListPage={initialThreadListPage}
        initialSession={initialSession}
        initialSettings={initialSettings}
        initialThreadList={initialThreadList}
        mode={threadListMode}
        showQuickShare={effectiveShowQuickShare}
        paginationBasePath={paginationBasePath}
      />
    </LStack>
  );
}

function ThreadListSection({
  initialSession,
  initialSettings,
  initialThreadList,
  initialThreadListPage,
  mode,
  showQuickShare,
  paginationBasePath,
}: {
  initialSession?: Account;
  initialSettings?: Settings;
  initialThreadList?: ThreadListResult;
  initialThreadListPage?: number;
  mode: "none" | "all" | "uncategorised";
  showQuickShare: boolean;
  paginationBasePath: string;
}) {
  const t = useTranslation();

  if (mode === "none") {
    return null;
  }

  const heading =
    mode === "all"
      ? t.category.allThreads
      : t.category.uncategorisedThreads;

  // Only show the category select when showing all threads, not uncategorised.
  const showCategorySelect = mode === "all";

  return (
    <LStack>
      {!showQuickShare && (
        <WStack>
          <Heading>{heading}</Heading>

          <ComposeAnchor />
        </WStack>
      )}

      <ThreadFeedScreen
        initialPage={initialThreadListPage}
        initialPageData={initialThreadList}
        initialSession={initialSession}
        initialSettings={initialSettings}
        category={mode === "all" ? undefined : null}
        paginationBasePath={paginationBasePath}
        showCategorySelect={showCategorySelect}
        showQuickShare={showQuickShare}
      />
    </LStack>
  );
}
