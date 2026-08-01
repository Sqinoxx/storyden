import { useNodeDraftList, useNodeList } from "@/api/openapi-client/nodes";
import { useThreadList } from "@/api/openapi-client/threads";
import {
  Account,
  NodeDraftListOKResponse,
  NodeListOKResponse,
  ThreadListOKResponse,
  Visibility,
} from "@/api/openapi-schema";

export type Props = {
  session: Account;
  initialThreads: ThreadListOKResponse;
  initialNodes: NodeListOKResponse;
  initialNodeDrafts?: NodeDraftListOKResponse;
};

export function useDraftListScreen({
  session,
  initialThreads,
  initialNodes,
  initialNodeDrafts,
}: Props) {
  const { data: threadsData, error: errorThreads } = useThreadList(
    {
      author: session.handle,
      visibility: [Visibility.draft, Visibility.review],
    },
    { swr: { fallbackData: initialThreads } },
  );

  const { data: nodesData, error: errorNodes } = useNodeList(
    {
      author: session.handle,
      visibility: [Visibility.draft, Visibility.review],
    },
    { swr: { fallbackData: initialNodes } },
  );

  const { data: nodeDraftsData, error: errorNodeDrafts } = useNodeDraftList(
    undefined,
    { swr: { fallbackData: initialNodeDrafts } },
  );

  if (!threadsData || !nodesData) {
    return {
      ready: false as const,
      error: errorThreads || errorNodes || errorNodeDrafts,
    };
  }

  const userNodeDrafts = (nodeDraftsData?.drafts ?? []).filter(
    (d) => d.author?.handle === session.handle,
  );

  const empty =
    initialNodes.nodes.length === 0 &&
    initialThreads.threads.length === 0 &&
    userNodeDrafts.length === 0;

  return {
    ready: true as const,
    empty,
    data: {
      nodes: nodesData.nodes,
      threads: threadsData.threads,
      nodeDrafts: userNodeDrafts,
    },

    session,
  };
}
