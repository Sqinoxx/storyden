import { useState } from "react";

import { handle } from "@/api/client";
import { threadSummarise } from "@/api/openapi-client/threads";
import { ThreadSummary } from "@/api/openapi-schema";

export const MIN_REPLIES_FOR_SUMMARY = 5;

export function useThreadSummary(slug: string) {
  const [summary, setSummary] = useState<ThreadSummary | null>(null);
  const [loading, setLoading] = useState(false);

  const summarise = async () => {
    setLoading(true);
    await handle(
      async () => {
        setSummary(await threadSummarise(slug));
      },
      {
        cleanup: async () => {
          setLoading(false);
        },
      },
    );
  };

  const close = () => setSummary(null);

  return { summary, loading, summarise, close };
}
