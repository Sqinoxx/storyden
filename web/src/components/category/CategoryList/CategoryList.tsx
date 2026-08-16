"use client";

import {
  TreeView as ArkTreeView,
  createTreeCollection,
} from "@ark-ui/react/tree-view";
import { useEffect, useMemo, useState } from "react";
import { KeyedMutator } from "swr";

import { handle } from "@/api/client";
import {
  categoryUpdatePosition,
  useCategoryList as useGetCategoryList,
} from "@/api/openapi-client/categories";
import {
  Category,
  CategoryListOKResponse,
  CategoryUpdatePositionBody,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { CategoryMenu } from "@/components/category/CategoryMenu/CategoryMenu";
import { Anchor } from "@/components/site/Anchor";
import { DiscussionRoute } from "@/components/site/Navigation/Anchors/Discussion";
import { NavigationHeader } from "@/components/site/Navigation/ContentNavigationList/NavigationHeader";
import { Unready } from "@/components/site/Unready";
import { BulletIcon } from "@/components/ui/icons/Bullet";
import { ChevronRightIcon } from "@/components/ui/icons/Chevron";
import { DiscussionIcon } from "@/components/ui/icons/Discussion";
import { useCategoryEvent, useEmitCategoryEvent } from "@/lib/category/events";
import { CategoryTree, buildCategoryTree } from "@/lib/category/tree";
import { css, cx } from "@/styled-system/css";
import { HStack, LStack } from "@/styled-system/jsx";
import { treeView } from "@/styled-system/recipes";
import { hasPermission } from "@/utils/permissions";
import { useTranslation } from "@/lib/i18n";

import { CategoryCreateTrigger } from "../CategoryCreate/CategoryCreateTrigger";

export type Props = {
  initialCategoryList?: CategoryListOKResponse;
  currentCategorySlug?: string;
};

export function CategoryList({
  initialCategoryList,
  currentCategorySlug,
}: Props) {
  const { data, error, mutate } = useGetCategoryList({
    swr: { fallbackData: initialCategoryList },
  });

  if (!data) {
    return <Unready error={error} />;
  }

  return (
    <CategoryListTree
      categories={data.categories}
      currentCategorySlug={currentCategorySlug}
      mutate={mutate}
    />
  );
}

export function CategoryListTree({
  categories,
  currentCategorySlug,
  mutate,
}: {
  categories: Category[];
  currentCategorySlug?: string;
  mutate: KeyedMutator<CategoryListOKResponse>;
}) {
  const session = useSession();
  const t = useTranslation();

  const canManageCategories = hasPermission(session, "MANAGE_CATEGORIES");

  const tree = buildCategoryTree(categories);

  const collection = createTreeCollection<CategoryTree>({
    nodeToValue: (cat) => cat?.id ?? "",
    nodeToString: (cat) => cat?.slug ?? "",
    nodeToChildren: (cat) => cat?.children ?? [],
    rootNode: { children: tree } as CategoryTree,
  });

  const rootNodes = collection.rootNode.children;

  const defaultExpanded = useMemo(() => {
    return rootNodes.map((node) => node.id);
  }, [rootNodes]);

  const [expandedValue, setExpandedValue] = useState<string[]>(defaultExpanded);
  const [isSectionCollapsed, setIsSectionCollapsed] = useState(false);

  useEffect(() => {
    try {
      const stored = localStorage.getItem("nav_section_collapsed_categories");
      if (stored !== null) {
        setIsSectionCollapsed(stored === "true");
      }
    } catch { }
  }, []);

  const handleToggleSectionCollapse = () => {
    setIsSectionCollapsed((prev) => {
      const next = !prev;
      try {
        localStorage.setItem("nav_section_collapsed_categories", String(next));
      } catch { }
      return next;
    });
  };

  const handleExpandedChange = (details: { expandedValue: string[] }) => {
    setExpandedValue(details.expandedValue);
  };

  const emitCategoryEvent = useEmitCategoryEvent();

  useCategoryEvent(
    "category:reorder-category",
    async ({ categorySlug, direction, newParent, targetCategory }) => {
      await handle(async () => {
        const params: CategoryUpdatePositionBody = (() => {
          switch (direction) {
            case "above":
              return {
                before: targetCategory,
                parent: newParent,
              };

            case "below":
              return {
                after: targetCategory,
                parent: newParent,
              };

            case "inside":
              return {
                parent: targetCategory,
              };
          }
        })();

        const response = await categoryUpdatePosition(
          encodeURIComponent(categorySlug),
          params,
        );
        await mutate(response, { revalidate: false });
      });
    },
  );

  if (!tree) {
    return <Unready />;
  }

  const styles = treeView();

  return (
    <LStack gap="0">
      <NavigationHeader
        href={DiscussionRoute}
        size="sm"
        controls={canManageCategories && <CategoryCreateTrigger hideLabel />}
      >
        <HStack gap="2">
          <DiscussionIcon w="5" h="5" />
          {t.category.home}
        </HStack>
      </NavigationHeader>

      <ArkTreeView.Root
        className={styles.root}
        collection={collection}
        expandedValue={expandedValue}
        onExpandedChange={handleExpandedChange}
      >
        <ArkTreeView.Tree className={styles.tree}>
          {rootNodes.map((cat, index) => (
            <CategoryTreeNode
              key={cat.id}
              currentCategorySlug={currentCategorySlug}
              parent={null}
              category={cat}
              styles={styles}
              indexPath={[]}
              siblings={rootNodes}
              siblingIndex={index}
              canManageCategories={canManageCategories}
              emitCategoryEvent={emitCategoryEvent}
            />
          ))}
        </ArkTreeView.Tree>
      </ArkTreeView.Root>
    </LStack>
  );
}

type TreeNodeProps = {
  currentCategorySlug: string | undefined;
  parent: CategoryTree | null;
  category: CategoryTree;
  styles: any;
  indexPath: number[];
  siblings: CategoryTree[];
  siblingIndex: number;
  canManageCategories: boolean;
  emitCategoryEvent: ReturnType<typeof useEmitCategoryEvent>;
};

function CategoryTreeNode({
  currentCategorySlug,
  parent,
  category,
  styles,
  indexPath,
  siblings,
  siblingIndex,
  canManageCategories,
  emitCategoryEvent,
}: TreeNodeProps) {
  const isHighlighted = category.slug === currentCategorySlug;

  const highlightStyles = css({
    background: isHighlighted ? "bg.selected" : undefined,
  });

  const previousSibling = siblings[siblingIndex - 1];
  const nextSibling = siblings[siblingIndex + 1];

  const onMoveUp = previousSibling
    ? () => {
      emitCategoryEvent("category:reorder-category", {
        categorySlug: category.slug,
        targetCategory: previousSibling.id,
        direction: "above",
        newParent: parent?.id ?? null,
      });
    }
    : undefined;

  const onMoveDown = nextSibling
    ? () => {
      emitCategoryEvent("category:reorder-category", {
        categorySlug: category.slug,
        targetCategory: nextSibling.id,
        direction: "below",
        newParent: parent?.id ?? null,
      });
    }
    : undefined;

  const onPromote = parent
    ? () => {
      emitCategoryEvent("category:reorder-category", {
        categorySlug: category.slug,
        targetCategory: parent.id,
        direction: "above",
        newParent: parent.parent ?? null,
      });
    }
    : undefined;

  return (
    <ArkTreeView.NodeProvider
      key={category.id}
      node={category}
      indexPath={indexPath}
    >
      <ArkTreeView.Branch className={cx(styles.branch)}>
        <ArkTreeView.BranchControl
          className={cx("group", styles.branchControl, highlightStyles)}
        >
          <ArkTreeView.BranchIndicator className={styles.branchIndicator}>
            {category.children.length > 0 ? (
              <ChevronRightIcon />
            ) : (
              <BulletIcon />
            )}
          </ArkTreeView.BranchIndicator>

          <ArkTreeView.BranchText asChild className={styles.branchText}>
            <Anchor
              href={`/d/${category.slug}`}
              className={css({
                display: "flex",
                alignItems: "center",
                gap: "2",
                flex: "1",
                _hover: { textDecoration: "none" },
              })}
            >
              {category.name}
            </Anchor>
          </ArkTreeView.BranchText>

          {canManageCategories && (
            <HStack
              gap="1"
              opacity={{
                base: "0",
                _groupHover: "full",
              }}
            >
              <CategoryMenu
                category={category}
                onMoveUp={onMoveUp}
                onMoveDown={onMoveDown}
                onPromote={onPromote}
              />
            </HStack>
          )}
        </ArkTreeView.BranchControl>

        <ArkTreeView.BranchContent className={styles.branchContent}>
          {category.children.map((child, childIndex) => (
            <CategoryTreeNode
              key={child.id}
              currentCategorySlug={currentCategorySlug}
              parent={category}
              category={child}
              indexPath={[...indexPath, childIndex]}
              styles={styles}
              siblings={category.children}
              siblingIndex={childIndex}
              canManageCategories={canManageCategories}
              emitCategoryEvent={emitCategoryEvent}
            />
          ))}
        </ArkTreeView.BranchContent>
      </ArkTreeView.Branch>
    </ArkTreeView.NodeProvider>
  );
}
