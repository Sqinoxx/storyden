"use client";

import { useMemo } from "react";

import { useAccessPassList } from "@/api/openapi-client/access";
import { useCategoryList } from "@/api/openapi-client/categories";
import { AccessPass, Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { hasPermission } from "@/utils/permissions";

import { CategoryAccess, categoryAccess, lockedCategoryIDs } from "./access";

export function useAccessPasses() {
  const { data, error, mutate } = useAccessPassList();

  return {
    passes: data?.passes ?? [],
    ready: data !== undefined,
    error,
    mutate,
  };
}

// Mirrors the backend bypass so staff never see locks they are exempt from.
export function useCanBypassAccess() {
  const session = useSession();
  return hasPermission(
    session,
    Permission.ADMINISTRATOR,
    Permission.MANAGE_ACCESS,
    Permission.MANAGE_POSTS,
  );
}

export function useCategoryAccess(
  categoryID: string | undefined,
): CategoryAccess {
  const { passes } = useAccessPasses();
  const { data } = useCategoryList();
  const bypass = useCanBypassAccess();

  return useMemo(() => {
    if (!categoryID || passes.length === 0) return { state: "open" };
    return categoryAccess(passes, data?.categories ?? [], categoryID, bypass);
  }, [passes, data, categoryID, bypass]);
}

export function useLockedCategoryIDs(): Set<string> {
  const { passes } = useAccessPasses();
  const { data } = useCategoryList();
  const bypass = useCanBypassAccess();

  return useMemo(
    () => lockedCategoryIDs(passes, data?.categories ?? [], bypass),
    [passes, data, bypass],
  );
}

export function usePassesByID(ids: string[]): AccessPass[] {
  const { passes } = useAccessPasses();
  return useMemo(() => passes.filter((p) => ids.includes(p.id)), [passes, ids]);
}
