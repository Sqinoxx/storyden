"use client";

import { useThreadRelated } from "@/api/openapi-client/threads";
import { useTranslation } from "@/lib/i18n";
import { useAIFeature } from "@/lib/settings/ai-features";

import { SimilarThreadList } from "./SimilarThreadList";

export function RelatedThreads({ slug }: { slug: string }) {
  const t = useTranslation();
  const enabled = useAIFeature("similar_threads");
  const { data } = useThreadRelated(slug, {
    swr: { enabled, revalidateOnFocus: false },
  });

  if (!enabled || !data) return null;

  return (
    <SimilarThreadList
      title={t.thread.relatedTitle}
      items={data.items}
      testId="related-threads"
    />
  );
}
