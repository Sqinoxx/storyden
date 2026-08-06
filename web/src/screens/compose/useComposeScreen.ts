import { useEffect, useState } from "react";

import { threadGet } from "@/api/openapi-client/threads";
import { Thread } from "@/api/openapi-schema";

import { handle } from "@/api/client";

export type Props = { editing?: string };

export function useComposeScreen({ editing }: Props) {
  const [loadingDraft, setLoadingDraft] = useState(editing !== undefined);
  const [draft, setDraft] = useState<Thread | undefined>(undefined);

  useEffect(() => {
    if (editing === undefined) {
      setDraft(undefined);
      setLoadingDraft(false);
      return;
    }

    setLoadingDraft(true);
    handle(
      async () => {
        const thread = await threadGet(editing);
        setDraft(thread);
      },
      {
        cleanup: async () => setLoadingDraft(false),
      },
    );
  }, [editing]);

  return {
    loadingDraft,
    draft,
  };
}
