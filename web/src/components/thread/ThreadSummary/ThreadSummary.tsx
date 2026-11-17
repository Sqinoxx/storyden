"use client";

import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { CloseIcon } from "@/components/ui/icons/Close";
import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { plainText } from "@/components/post/AttachmentSummary/useAttachmentSummary";
import { useTranslation } from "@/lib/i18n";
import { useAIFeature } from "@/lib/settings/ai-features";
import { css } from "@/styled-system/css";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";

import { MIN_REPLIES_FOR_SUMMARY, useThreadSummary } from "./useThreadSummary";

type Props = {
  slug: string;
  replyCount: number;
};

const summaryContent = css({
  fontSize: "sm",
  "& p": { mt: "2" },
  "& p:first-child": { mt: "0" },
  "& ul": { listStyleType: "disc", pl: "5", mt: "1" },
  "& li": { mt: "1" },
});

export function ThreadSummary({ slug, replyCount }: Props) {
  const t = useTranslation();
  const enabled = useAIFeature("thread_summary");
  const { state, summarise, close } = useThreadSummary(slug);
  const loading = state.kind === "streaming" && !state.text;

  if (!enabled || replyCount < MIN_REPLIES_FOR_SUMMARY) return null;

  if (state.kind === "idle" || loading) {
    return (
      <HStack w="full" justify="end">
        <Button
          type="button"
          size="xs"
          variant="subtle"
          loading={loading}
          onClick={summarise}
        >
          <IntelligenceIcon />
          {loading ? t.thread.summarising : t.thread.summarise}
        </Button>
      </HStack>
    );
  }

  return (
    <LStack
      w="full"
      gap="2"
      p="3"
      borderRadius="md"
      borderWidth="thin"
      borderColor="border.subtle"
      bg="bg.muted"
    >
      <WStack>
        <HStack gap="2" color="fg.muted" fontSize="sm" fontWeight="medium">
          <IntelligenceIcon width="4" height="4" />
          {t.thread.summaryTitle}
        </HStack>
        <IconButton
          type="button"
          size="xs"
          variant="ghost"
          aria-label={t.common.close}
          onClick={close}
        >
          <CloseIcon />
        </IconButton>
      </WStack>

      {state.kind === "streaming" && (
        <styled.p
          fontSize="sm"
          whiteSpace="pre-line"
          data-testid="thread-summary-live"
        >
          {plainText(state.text)}
        </styled.p>
      )}

      {state.kind === "failed" && (
        <styled.p fontSize="sm" color="fg.destructive">
          {t.thread.summaryFailed}
        </styled.p>
      )}

      {state.kind === "done" &&
        (state.summary.summary ? (
          <div
            className={summaryContent}
            data-testid="thread-summary"
            dangerouslySetInnerHTML={{ __html: state.summary.summary }}
          />
        ) : (
          <styled.p fontSize="sm">{t.thread.summaryEmpty}</styled.p>
        ))}

      {state.kind !== "failed" && (
        <styled.p fontSize="xs" color="fg.subtle">
          {t.thread.summaryDisclaimer}
          {state.kind === "done" &&
            state.summary.includes_attachments &&
            ` ${t.thread.summaryWithAttachments}`}
        </styled.p>
      )}
    </LStack>
  );
}
