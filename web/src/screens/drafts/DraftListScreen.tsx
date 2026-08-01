"use client";

import { NodeCardRows } from "@/components/library/NodeCardList";
import { ThreadReferenceList } from "@/components/post/ThreadReferenceList";
import { QueueVersionList } from "@/components/queue/QueueVersionList";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { VStack } from "@/styled-system/jsx";

import { useLibraryPath } from "../library/useLibraryPath";

import { Props, useDraftListScreen } from "./useDraftListScreen";

export function DraftListScreen(props: Props) {
  const { ready, data, error } = useDraftListScreen(props);
  const libraryPath = useLibraryPath();

  if (!ready) return <Unready error={error} />;

  const { nodes, threads, nodeDrafts } = data;

  return (
    <VStack w="full" alignItems="start" gap="4">
      <Heading>Your drafts</Heading>

      {threads.length > 0 && (
        <>
          <Heading color="fg.subtle">Threads</Heading>
          <ThreadReferenceList threads={threads} />
        </>
      )}

      {(nodes.length > 0 || (nodeDrafts && nodeDrafts.length > 0)) && (
        <>
          <Heading color="fg.subtle">Library</Heading>
          {nodes.length > 0 && (
            <NodeCardRows
              libraryPath={libraryPath}
              context="generic"
              nodes={nodes}
            />
          )}
          {nodeDrafts && nodeDrafts.length > 0 && (
            <QueueVersionList drafts={nodeDrafts} />
          )}
        </>
      )}
    </VStack>
  );
}
