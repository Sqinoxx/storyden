import { useMemo } from "react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import {
  CategoryTree,
  buildCategoryTree,
  findCategoryNode,
  isLeafCategory,
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

  const tree = useMemo(() => {
    if (!data) {
      return [];
    }

    const full = buildCategoryTree(data.categories);

    if (!rootId) {
      return full;
    }

    return findCategoryNode(full, rootId)?.children ?? full;
  }, [data, rootId]);

  const isSelectable = useMemo(() => {
    if (selectable === "all" || canPostAnywhere) {
      return () => true;
    }

    return (node: CategoryTree) => isLeafCategory(node);
  }, [selectable, canPostAnywhere]);

  return {
    ready: data !== undefined,
    error,
    tree,
    canPostAnywhere,
    isSelectable,
  };
}
