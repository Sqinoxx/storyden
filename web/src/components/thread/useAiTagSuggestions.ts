import { useEffect, useRef, useState } from "react";

import { threadSuggestTags } from "@/api/openapi-client/threads";

const DEBOUNCE_MS = 2000;
const MIN_CHARS = 20;

// Suggestions accumulate rather than being replaced so a tag the member removed
// stays matched, which keeps it from being re-added when a later request
// briefly omits and then returns it again.
export function useAiTagSuggestions(
  title: string | undefined,
  body: string | undefined,
  enabled: boolean,
) {
  const [tags, setTags] = useState<Set<string>>(new Set());
  const requestRef = useRef(0);

  useEffect(() => {
    if (!enabled) return;

    const plain = `${title ?? ""} ${(body ?? "").replace(/<[^>]*>/g, " ")}`;
    if (plain.replace(/\s+/g, "").length < MIN_CHARS) return;

    const request = ++requestRef.current;
    const timer = setTimeout(async () => {
      try {
        const res = await threadSuggestTags({ title, body });
        if (request !== requestRef.current || res.tags.length === 0) return;

        setTags((prev) => {
          const next = new Set(prev);
          for (const t of res.tags) next.add(t.toLowerCase());
          return next.size === prev.size ? prev : next;
        });
      } catch {
        // Suggestions are best-effort; the editor works without them.
      }
    }, DEBOUNCE_MS);

    return () => clearTimeout(timer);
  }, [title, body, enabled]);

  return tags;
}
