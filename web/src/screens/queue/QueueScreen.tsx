"use client";

import { useNodeDraftList, useNodeList } from "@/api/openapi-client/nodes";
import { Visibility } from "@/api/openapi-schema";
import { QueueNodeList } from "@/components/queue/QueueNodeList";
import { QueueVersionList } from "@/components/queue/QueueVersionList";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { LStack } from "@/styled-system/jsx";

export function QueueScreen() {
  const t = useTranslation();
  const { data: submissions, error: submissionError } = useNodeList({
    visibility: [Visibility.review],
    format: "flat",
  });
  const { data: drafts, error: draftsError } = useNodeDraftList();

  if (!submissions || !drafts) {
    return <Unready error={submissionError ?? draftsError} />;
  }

  return (
    <LStack gap="8">
      <LStack>
        <Heading>{t.queue.submissions}</Heading>

        <QueueNodeList nodes={submissions.nodes} />
      </LStack>

      <LStack>
        <Heading>{t.queue.pageEdits}</Heading>

        <QueueVersionList drafts={drafts.drafts} />
      </LStack>
    </LStack>
  );
}
