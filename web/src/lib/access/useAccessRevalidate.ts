"use client";

import { useSWRConfig } from "swr";

// Grants change both the manager lists and every member-facing lock, so any
// write revalidates everything under /access at once.
export function useAccessRevalidate() {
  const { mutate } = useSWRConfig();

  return () =>
    mutate(
      (key) =>
        Array.isArray(key) &&
        typeof key[0] === "string" &&
        key[0].startsWith("/access"),
    );
}
