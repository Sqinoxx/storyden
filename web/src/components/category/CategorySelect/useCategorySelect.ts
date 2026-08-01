import { createListCollection } from "@ark-ui/react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { ListCollectionItem } from "@/components/ui/form/SelectField";
import { useSession } from "@/auth";
import { Permission } from "@/api/openapi-schema";
import { hasPermission } from "@/utils/permissions";

export const NO_CATEGORY_VALUE = "__none__";

export function useCategorySelect() {
  const session = useSession();
  const canManageCategories = hasPermission(session, Permission.MANAGE_CATEGORIES);
  const { data, error } = useCategoryList();

  if (!data) {
    if (error) {
      console.error("Failed to load categories", error);
    }

    // An empty collection because we don't want to show an errored/empty state.
    const collection = createListCollection<ListCollectionItem>({ items: [] });

    return {
      ready: false as const,
      collection,
      error,
    };
  }

  // Identify all category IDs that act as parent categories
  const parentIds = new Set<string>();
  data.categories.forEach((cat) => {
    if (cat.children && cat.children.length > 0) {
      parentIds.add(cat.id);
    }
    if (cat.parent) {
      const pid = typeof cat.parent === "string" ? cat.parent : (cat.parent as { id?: string }).id;
      if (pid) {
        parentIds.add(pid);
      }
    }
  });

  let categoriesToDisplay = data.categories;
  if (!canManageCategories) {
    categoriesToDisplay = categoriesToDisplay.filter((cat) => !parentIds.has(cat.id));
  }

  const items: ListCollectionItem[] = [];

  if (canManageCategories) {
    items.push({ label: "No category", value: NO_CATEGORY_VALUE });
  }

  items.push(
    ...categoriesToDisplay.map((category) => ({
      label: category.name,
      value: category.id,
    })),
  );

  const collection = createListCollection({
    items,
  });

  return {
    ready: true as const,
    collection,
  };
}

