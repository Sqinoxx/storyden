"use client";

import { useMemo } from "react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { useDriveFolderList } from "@/api/openapi-client/drive";
import { AccessPass } from "@/api/openapi-schema";

// Human readable list of what a pass unlocks: categories first, then Drive
// folders, skipping anything the viewer cannot see.
export function useIncludedNames() {
  const { data: categories } = useCategoryList();
  const { data: drive } = useDriveFolderList();

  return useMemo(() => {
    const names = new Map<string, string>();
    (categories?.categories ?? []).forEach((c) => names.set(c.id, c.name));
    (drive?.folders ?? []).forEach((f) => names.set(f.id, f.name));

    return (pass: AccessPass) =>
      [...pass.category_ids, ...pass.drive_folder_ids]
        .map((id) => names.get(id))
        .filter((n): n is string => Boolean(n));
  }, [categories, drive]);
}
