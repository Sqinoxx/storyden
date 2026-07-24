"use client";

import { EmptyState } from "../site/EmptyState";
import { EmptyThreadsIcon } from "../ui/icons/Empty";
import { useTranslation } from "@/lib/i18n";

export function FeedEmptyState() {
  const t = useTranslation();
  return (
    <EmptyState w="full" icon={<EmptyThreadsIcon />}>
      <p>{t.feed.empty}</p>
    </EmptyState>
  );
}
