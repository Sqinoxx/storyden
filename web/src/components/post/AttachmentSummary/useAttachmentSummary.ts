import { useCallback, useEffect, useRef, useState } from "react";

import { handle } from "@/api/client";
import { threadAttachmentSummaryGet } from "@/api/openapi-client/threads";
import { AttachmentSummary } from "@/api/openapi-schema";
import { API_ADDRESS } from "@/config";

export type SummaryState =
  | { kind: "idle" }
  | { kind: "loading" }
  | { kind: "unavailable" }
  | { kind: "streaming"; preview: string; text: string }
  | { kind: "done"; summary: string; quality?: string }
  | { kind: "failed"; preview?: string };

const tags = /<[^>]*>/g;

export function plainText(html: string) {
  return html
    .replace(/<\/li>/g, "\n")
    .replace(tags, "")
    .replace(/\n{2,}/g, "\n");
}

export function useAttachmentSummary(slug: string, assetID: string) {
  const [state, setState] = useState<SummaryState>({ kind: "idle" });
  const source = useRef<EventSource | null>(null);

  const stop = useCallback(() => {
    source.current?.close();
    source.current = null;
  }, []);

  useEffect(() => stop, [stop]);

  const stream = useCallback(
    (preview: string) => {
      stop();
      setState({ kind: "streaming", preview, text: "" });

      const url = `${API_ADDRESS}/api/threads/${encodeURIComponent(slug)}/attachments/${encodeURIComponent(assetID)}/summary/stream`;
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
          const { summary } = JSON.parse((e as MessageEvent).data) as {
            summary: string;
          };
          setState({ kind: "done", summary, quality: "quick" });
        } catch {
          setState({ kind: "failed", preview });
        }
      });

      es.addEventListener("error", () => {
        stop();
        setState((s) => (s.kind === "done" ? s : { kind: "failed", preview }));
      });
    },
    [slug, assetID, stop],
  );

  const open = useCallback(async () => {
    setState({ kind: "loading" });
    await handle(
      async () => {
        const res: AttachmentSummary = await threadAttachmentSummaryGet(
          slug,
          assetID,
        );
        switch (res.status) {
          case "ready":
            setState({
              kind: "done",
              summary: res.summary ?? "",
              quality: res.quality,
            });
            return;
          case "unavailable":
            setState({ kind: "unavailable" });
            return;
          default:
            stream(res.preview);
        }
      },
      {
        errorToast: false,
        onError: async () => setState({ kind: "failed" }),
      },
    );
  }, [slug, assetID, stream]);

  const close = useCallback(() => {
    stop();
    setState({ kind: "idle" });
  }, [stop]);

  return { state, open, close };
}
