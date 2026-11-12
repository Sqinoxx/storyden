"use client";

import { MenuSelectionDetails, Portal } from "@ark-ui/react";

import { Category, Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { MoreAction } from "@/components/site/Action/More";
import { ArrowDownIcon, ArrowUpIcon, PromoteCategoryIcon } from "@/components/ui/icons/Arrow";
import { CategoryIcon } from "@/components/ui/icons/Category";
import { SubmenuIcon } from "@/components/ui/icons/Submenu";
import { Heading } from "@/components/ui/heading";
import * as Menu from "@/components/ui/menu";
import { WEB_ADDRESS } from "@/config";
import { CategoryTree } from "@/lib/category/tree";
import { HStack, styled } from "@/styled-system/jsx";
import { useShare } from "@/utils/client";
import { hasPermission } from "@/utils/permissions";
import { useCopyToClipboard } from "@/utils/useCopyToClipboard";

import { CategoryBadge } from "../CategoryBadge";
import { CategoryCreateMenuItem } from "../CategoryCreate/CategoryCreateMenuItem";
import { CategoryDeleteMenuItem } from "../CategoryDelete/CategoryDeleteMenuItem";
import { CategoryEditMenuItem } from "../CategoryEdit/CategoryEdit";
import { useTranslation } from "@/lib/i18n";

type Props = {
  category: Category;
  onMoveUp?: () => void;
  onMoveDown?: () => void;
  onPromote?: () => void;
  moveTargets?: CategoryTree[];
  onMoveInto?: (targetCategory: string) => void;
};

const MOVE_INTO_PREFIX = "move-into:";

function excludeSubtree(
  nodes: CategoryTree[],
  excludedID: string,
): CategoryTree[] {
  return nodes
    .filter((node) => node.id !== excludedID)
    .map((node) => ({
      ...node,
      children: excludeSubtree(node.children, excludedID),
    }));
}

export function useCategoryMenu({
  category,
  onMoveUp,
  onMoveDown,
  onPromote,
  onMoveInto,
}: Props) {
  const t = useTranslation();
  const account = useSession();
  const [, copyToClipboard] = useCopyToClipboard();

  const isEditingEnabled = hasPermission(account, Permission.MANAGE_CATEGORIES);

  const isSharingEnabled = useShare();

  const permalink = `${WEB_ADDRESS}/d/${category.slug}`;

  async function handleCopyLink() {
    await copyToClipboard(permalink);
  }

  async function handleShare() {
    await navigator.share({
      title: `${t.category.discussionCategory}: ${category.name}`,
      url: permalink,
      text: category.description,
    });
  }

  async function handleSelect({ value }: MenuSelectionDetails) {
    if (value.startsWith(MOVE_INTO_PREFIX)) {
      return onMoveInto?.(value.slice(MOVE_INTO_PREFIX.length));
    }

    switch (value) {
      case "copy-link":
        return handleCopyLink();

      case "share":
        return handleShare();

      case "edit":
        // Handled by item component
        return;

      case "create-subcategory":
        // Handled by item component
        return;

      case "delete":
        // Handled by item component
        return;

      case "move-up":
        return onMoveUp?.();

      case "move-down":
        return onMoveDown?.();

      case "promote":
        return onPromote?.();
    }
  }

  return {
    isSharingEnabled,
    isEditingEnabled,
    handlers: {
      handleSelect,
    },
  };
}

export function CategoryMenu(props: Props) {
  const { isSharingEnabled, isEditingEnabled, handlers } =
    useCategoryMenu(props);
  const t = useTranslation();

  const { category, onMoveUp, onMoveDown, onPromote, onMoveInto } = props;

  const moveTargets =
    onMoveInto && props.moveTargets
      ? excludeSubtree(props.moveTargets, category.id)
      : [];
  const canMoveInto = moveTargets.length > 0;

  const hasReorderActions =
    isEditingEnabled && (onMoveUp || onMoveDown || onPromote || canMoveInto);

  return (
    <Menu.Root lazyMount onSelect={handlers.handleSelect}>
      <Menu.Trigger asChild>
        <MoreAction size="xs" />
      </Menu.Trigger>

      <Portal>
        <Menu.Positioner>
          <Menu.Content minW="48" userSelect="none">
            <Menu.ItemGroup id="account">
              <Menu.ItemGroupLabel>
                <Heading size="sm">{category.name}</Heading>
                <styled.span color="fg.subtle">{t.category.discussionCategory}</styled.span>
              </Menu.ItemGroupLabel>

              <Menu.Separator />

              <Menu.Item value="copy-link">{t.actions.copyLink}</Menu.Item>
              {isSharingEnabled && <Menu.Item value="share">{t.actions.share}</Menu.Item>}
            </Menu.ItemGroup>

            {hasReorderActions && (
              <>
                <Menu.Separator />

                <Menu.ItemGroup id="reorder">
                  {onMoveUp && (
                    <Menu.Item value="move-up">
                      <HStack gap="1">
                        <ArrowUpIcon />
                        {t.category.moveUp}
                      </HStack>
                    </Menu.Item>
                  )}
                  {onMoveDown && (
                    <Menu.Item value="move-down">
                      <HStack gap="1">
                        <ArrowDownIcon />
                        {t.category.moveDown}
                      </HStack>
                    </Menu.Item>
                  )}
                  {onPromote && (
                    <Menu.Item value="promote">
                      <HStack gap="1">
                        <PromoteCategoryIcon />
                        {t.category.promoteToParentLevel}
                      </HStack>
                    </Menu.Item>
                  )}
                  {canMoveInto && (
                    <Menu.Root
                      lazyMount
                      size="xs"
                      positioning={{ placement: "right-start", gutter: -2 }}
                      onSelect={handlers.handleSelect}
                    >
                      <Menu.TriggerItem justifyContent="space-between">
                        <HStack gap="1">
                          <CategoryIcon />
                          {t.category.moveIntoCategory}
                        </HStack>
                        <SubmenuIcon />
                      </Menu.TriggerItem>

                      <Portal>
                        <Menu.Positioner>
                          <Menu.Content minW="48" userSelect="none">
                            {moveTargets.map((node) => (
                              <CategoryMoveIntoNode
                                key={node.id}
                                node={node}
                                currentParentID={category.parent}
                                onSelect={handlers.handleSelect}
                              />
                            ))}
                          </Menu.Content>
                        </Menu.Positioner>
                      </Portal>
                    </Menu.Root>
                  )}
                </Menu.ItemGroup>
              </>
            )}

            {isEditingEnabled && (
              <>
                <Menu.Separator />

                <Menu.ItemGroup id="manage">
                  <CategoryCreateMenuItem parentCategory={category} />
                  <CategoryEditMenuItem {...category} />
                  <CategoryDeleteMenuItem {...category} />
                </Menu.ItemGroup>
              </>
            )}
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}

function CategoryMoveIntoNode({
  node,
  currentParentID,
  onSelect,
}: {
  node: CategoryTree;
  currentParentID: string | undefined;
  onSelect: (details: MenuSelectionDetails) => void;
}) {
  const item = (
    <Menu.Item
      value={`${MOVE_INTO_PREFIX}${node.id}`}
      disabled={node.id === currentParentID}
    >
      <CategoryBadge category={node} asLink={false} />
    </Menu.Item>
  );

  if (node.children.length === 0) {
    return item;
  }

  return (
    <Menu.Root
      lazyMount
      size="xs"
      positioning={{ placement: "right-start", gutter: -2 }}
      onSelect={onSelect}
    >
      <Menu.TriggerItem justifyContent="space-between">
        <CategoryBadge category={node} asLink={false} />
        <SubmenuIcon />
      </Menu.TriggerItem>

      <Portal>
        <Menu.Positioner>
          <Menu.Content minW="48" userSelect="none">
            {item}
            <Menu.Separator />
            {node.children.map((child) => (
              <CategoryMoveIntoNode
                key={child.id}
                node={child}
                currentParentID={currentParentID}
                onSelect={onSelect}
              />
            ))}
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}
