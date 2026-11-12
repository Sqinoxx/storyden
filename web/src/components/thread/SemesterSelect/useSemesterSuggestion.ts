import { useEffect, useMemo, useState } from "react";
import { Control, FieldValues, Path, useWatch } from "react-hook-form";

import { threadSuggestSemester } from "@/api/openapi-client/threads";
import { Asset } from "@/api/openapi-schema";
import { parseTermKey, Term } from "@/lib/thread/semester";

const DEBOUNCE_MS = 1200;

export type SemesterSuggestion = {
  key: string;
  term: Term;
  evidence: string;
};

type Props<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  attachments: Asset[];
};

export function useSemesterSuggestion<T extends FieldValues>({
  control,
  name,
  attachments,
}: Props<T>) {
  const title = useWatch({ control, name: "title" as Path<T> }) as
    | string
    | undefined;
  const body = useWatch({ control, name: "body" as Path<T> }) as
    | string
    | undefined;
  const selected = useWatch({ control, name }) as string | undefined;

  const filenames = useMemo(
    () => attachments.map((a) => a.filename).filter(Boolean),
    [attachments],
  );
  const filenamesKey = filenames.join("\n");

  const [suggestion, setSuggestion] = useState<SemesterSuggestion | null>(
    null,
  );
  const [dismissed, setDismissed] = useState<Set<string>>(new Set());

  useEffect(() => {
    if (!title?.trim() && !body?.trim() && filenames.length === 0) {
      setSuggestion(null);
      return;
    }

    let cancelled = false;
    const timer = setTimeout(async () => {
      try {
        const res = await threadSuggestSemester({
          title,
          body,
          filenames,
        });
        if (cancelled) return;

        const term = parseTermKey(res.term);
        setSuggestion(
          term && res.term
            ? { key: res.term, term, evidence: res.evidence ?? "" }
            : null,
        );
      } catch {
        if (!cancelled) setSuggestion(null);
      }
    }, DEBOUNCE_MS);

    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [title, body, filenamesKey]);

  const visible =
    suggestion &&
    suggestion.key !== selected &&
    !dismissed.has(suggestion.key)
      ? suggestion
      : null;

  const dismiss = () => {
    if (!suggestion) return;
    setDismissed((prev) => new Set(prev).add(suggestion.key));
  };

  return { suggestion: visible, dismiss };
}
