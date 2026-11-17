import { useCallback, useEffect, useRef, useState } from "react";

import { ThreadSummary } from "@/api/openapi-schema";
import { API_ADDRESS } from "@/config";

export const MIN_REPLIES_FOR_SUMMARY = 5;

export type ThreadSummaryState =
  | { kind: "idle" }
  | { kind: "streaming"; text: string }
  | { kind: "done"; summary: ThreadSummary }
  | { kind: "failed" };

export function useThreadSummary(slug: string) {
  const [state, setState] = useState<ThreadSummaryState>({ kind: "idle" });
  const source = useRef<EventSource | null>(null);

  const stop = useCallback(() => {
    source.current?.close();
    source.current = null;
  }, []);

  useEffect(() => stop, [stop]);

  const summarise = useCallback(() => {
    stop();
    setState({ kind: "streaming", text: "" });

    const url = `${API_ADDRESS}/api/threads/${encodeURIComponent(slug)}/summary/stream`;
    const es = new EventSource(url, { withCredentials: true });
    source.current = es;

    es.onmessage = (e) => {
      try {
        const { t } = JSON.parse(e.data) as { t: string };
        setState((s) =>
          s.kind === "streaming" ? { ...s, text: s.text + t } : s,
        );
      } catch {}
    };

    es.addEventListener("done", (e) => {
      stop();
      try {
        const summary = JSON.parse((e as MessageEvent).data) as ThreadSummary;
        setState({ kind: "done", summary });
      } catch {
        setState({ kind: "failed" });
      }
    });

    es.addEventListener("error", () => {
      stop();
      setState((s) => (s.kind === "done" ? s : { kind: "failed" }));
    });
  }, [slug, stop]);

  const close = useCallback(() => {
    stop();
    setState({ kind: "idle" });
  }, [stop]);

  return { state, summarise, close };
}
