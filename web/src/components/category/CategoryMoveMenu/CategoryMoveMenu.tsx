import { MenuSelectionDetails, Portal } from "@ark-ui/react";

import { handle } from "@/api/client";
import { useCategoryList } from "@/api/openapi-client/categories";
import { Permission, ThreadReference } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { Unready } from "@/components/site/Unready";
import { CategoryIcon } from "@/components/ui/icons/Category";
import { SubmenuIcon } from "@/components/ui/icons/Submenu";
import * as Menu from "@/components/ui/menu";
import { CategoryTree, buildCategoryTree } from "@/lib/category/tree";
import { useThreadMutations } from "@/lib/thread/mutation";
import { HStack } from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";

import { CategoryBadge } from "../CategoryBadge";
import {
  CategoryCreateTrigger,
  CreateCategoryID,
  CreateCategoryMenuItem,
} from "../CategoryCreate/CategoryCreateTrigger";

type Props = {
  thread: ThreadReference;
};

export function useCategoryMoveMenu({ thread }: Props) {
  const { revalidate, updateCategory } = useThreadMutations(thread);

  async function handleSelect({ value }: MenuSelectionDetails) {
    // If the user clicked the create category prompt, only present when there
    // are no categories available, then exit early.
    if (value === CreateCategoryID) {
      return;
    }

    await handle(
      async () => {
        await updateCategory(value);
      },
      {
        promiseToast: {
          loading: "Moving thread...",
          success: "Moved!",
        },
        async cleanup() {
          await revalidate();
        },
      },
    );
  }

  return {
    handlers: {
      handleSelect,
    },
  };
}

export function CategoryMoveMenu(props: Props) {
  const { handlers } = useCategoryMoveMenu(props);

  return (
    <Menu.Root
      size="xs"
      //   lazyMount
      positioning={{ placement: "right-start", gutter: -2 }}
      onSelect={handlers.handleSelect}
    >
      <Menu.TriggerItem justifyContent="space-between">
        <HStack gap="1">
          <CategoryIcon />
          Move
        </HStack>
        <SubmenuIcon />
      </Menu.TriggerItem>

      <Portal>
        <Menu.Positioner>
          <LazyLoadedCategoryMoveMenuContent
            onSelect={handlers.handleSelect}
          />
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}

function LazyLoadedCategoryMoveMenuContent({
  onSelect,
}: {
  onSelect: (details: MenuSelectionDetails) => void;
}) {
  const session = useSession();
  const { data, error } = useCategoryList();

  if (!data) {
    return <Unready error={error} />;
  }

  const { categories } = data;

  const hasAnyCategories = categories.length > 0;
  if (!hasAnyCategories) {
    return (
      <Menu.Content minW="48" userSelect="none">
        <Menu.ItemGroup id="move-no-categories">
          <Menu.ItemGroupLabel>No categories to move to</Menu.ItemGroupLabel>

          <CreateCategoryMenuItem />
        </Menu.ItemGroup>
      </Menu.Content>
    );
  }

  const tree = buildCategoryTree(categories);
  const canMoveAnywhere = hasPermission(
    session,
    Permission.POST_IN_ANY_CATEGORY,
  );

  return (
    <Menu.Content minW="48" userSelect="none">
      <Menu.ItemGroup id="move">
        <Menu.ItemGroupLabel>Move thread</Menu.ItemGroupLabel>

        <Menu.Separator />

        {tree.map((node) => (
          <CategoryMoveMenuNode
            key={node.id}
            node={node}
            canMoveAnywhere={canMoveAnywhere}
            onSelect={onSelect}
          />
        ))}
      </Menu.ItemGroup>
    </Menu.Content>
  );
}

function CategoryMoveMenuNode({
  node,
  canMoveAnywhere,
  onSelect,
}: {
  node: CategoryTree;
  canMoveAnywhere: boolean;
  onSelect: (details: MenuSelectionDetails) => void;
}) {
  if (node.children.length === 0) {
    return (
      <Menu.Item value={node.id}>
        <CategoryBadge category={node} asLink={false} />
      </Menu.Item>
    );
  }

  return (
    <Menu.Root
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
            {canMoveAnywhere && (
              <>
                <Menu.Item value={node.id}>
                  <CategoryBadge category={node} asLink={false} />
                </Menu.Item>
                <Menu.Separator />
              </>
            )}

            {node.children.map((child) => (
              <CategoryMoveMenuNode
                key={child.id}
                node={child}
                canMoveAnywhere={canMoveAnywhere}
                onSelect={onSelect}
              />
            ))}
          </Menu.Content>
        </Menu.Positioner>
      </Portal>
    </Menu.Root>
  );
}
