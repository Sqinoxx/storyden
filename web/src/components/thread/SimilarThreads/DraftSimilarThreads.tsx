"use client";

import { useEffect, useRef, useState } from "react";
import { useFormContext, useWatch } from "react-hook-form";

import { threadSimilar } from "@/api/openapi-client/threads";
import { DatagraphItem } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import { useAIFeature } from "@/lib/settings/ai-features";

import { SimilarThreadList } from "./SimilarThreadList";

const DEBOUNCE_MS = 2000;
const MIN_CHARS = 20;

export function DraftSimilarThreads() {
  const t = useTranslation();
  const enabled = useAIFeature("similar_threads");
  const { control } = useFormContext();
  const title = useWatch({ control, name: "title" }) as string | undefined;
  const body = useWatch({ control, name: "body" }) as string | undefined;

  const [items, setItems] = useState<DatagraphItem[]>([]);
  const requestRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    const plain = `${title ?? ""} ${(body ?? "").replace(/<[^>]*>/g, " ")}`;
    if (plain.replace(/\s+/g, "").length < MIN_CHARS) return;

    const request = ++requestRef.current;
    const timer = setTimeout(async () => {
      try {
        const res = await threadSimilar({ title, body });
        if (request === requestRef.current) setItems(res.items);
      } catch {
        // Hints are best-effort; composing works without them.
      }
    }, DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [title, body, enabled]);

  if (!enabled) return null;

  return (
    <SimilarThreadList
      title={t.thread.similarDraftTitle}
      hint={t.thread.similarDraftHint}
      items={items}
      testId="draft-similar-threads"
    />
  );
}
