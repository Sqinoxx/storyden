"use client";

import { NodeCardRows } from "@/components/library/NodeCardList";
import { ThreadReferenceList } from "@/components/post/ThreadReferenceList";
import { QueueVersionList } from "@/components/queue/QueueVersionList";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { VStack } from "@/styled-system/jsx";

import { useLibraryPath } from "../library/useLibraryPath";

import { Props, useDraftListScreen } from "./useDraftListScreen";

export function DraftListScreen(props: Props) {
  const t = useTranslation();
  const { ready, data, error } = useDraftListScreen(props);
  const libraryPath = useLibraryPath();

  if (!ready) return <Unready error={error} />;

  const { nodes, threads, nodeDrafts } = data;

  return (
    <VStack w="full" alignItems="start" gap="4">
      <Heading>{t.drafts.title}</Heading>

      {threads.length > 0 && (
        <>
          <Heading color="fg.subtle">{t.profile.threads}</Heading>
          <ThreadReferenceList threads={threads} />
        </>
      )}

      {(nodes.length > 0 || (nodeDrafts && nodeDrafts.length > 0)) && (
        <>
          <Heading color="fg.subtle">{t.nav.library}</Heading>
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
