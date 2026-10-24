import { useMemo } from "react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import {
  CategoryTree,
  buildCategoryTree,
  categoryDepths,
  findCategoryNode,
  isMemberPostableCategory,
} from "@/lib/category/tree";
import { hasPermission } from "@/utils/permissions";

export const NO_CATEGORY_VALUE = "__none__";

export type SelectableMode = "leaf" | "all";

export type UseCategoryTreeSelectProps = {
  rootId?: string;
  selectable?: SelectableMode;
};

export function useCategoryTreeSelect({
  rootId,
  selectable = "leaf",
}: UseCategoryTreeSelectProps) {
  const session = useSession();
  const { data, error } = useCategoryList();

  const canPostAnywhere = hasPermission(
    session,
    Permission.POST_IN_ANY_CATEGORY,
  );

  const full = useMemo(
    () => (data ? buildCategoryTree(data.categories) : []),
    [data],
  );

  const isSelectable = useMemo(() => {
    if (selectable === "all" || canPostAnywhere) {
      return () => true;
    }

    const depths = categoryDepths(full);

    return (node: CategoryTree) =>
      isMemberPostableCategory(
        depths.get(node.id) ?? 0,
        node.children.length > 0,
      );
  }, [full, selectable, canPostAnywhere]);

  const tree = useMemo(() => {
    if (!rootId) {
      return full;
    }

    const rootNode = findCategoryNode(full, rootId);
    if (!rootNode) {
      return full;
    }

    // When the viewed category itself may be posted into, it must be
    // available to pick, not just its children - otherwise the user is forced
    // into a subcategory despite being allowed to post directly into it.
    if (isSelectable(rootNode)) {
      return [rootNode];
    }

    return rootNode.children;
  }, [full, rootId, isSelectable]);

  return {
    ready: data !== undefined,
    error,
    tree,
    canPostAnywhere,
    isSelectable,
  };
}
