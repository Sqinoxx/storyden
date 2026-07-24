"use client";

import { DatagraphItemKind, ProfileReference } from "@/api/openapi-schema";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { VStack, styled } from "@/styled-system/jsx";

import { ReportModal, ReportModalProps } from "./ReportModal";
import { useTranslation } from "@/lib/i18n";

type Props = Omit<
  ReportModalProps,
  | "title"
  | "description"
  | "subject"
  | "targetKind"
  | "targetId"
  | "submitLabel"
  | "successMessage"
  | "loadingMessage"
> & {
  targetKind: "thread" | "reply" | "post";
  targetId: string;
  author: ProfileReference;
  headline?: string;
  body?: string;
};

export function ReportPostModal({
  targetKind,
  targetId,
  author,
  headline,
  body,
  ...disclosure
}: Props) {
  const t = useTranslation();

  const modalTitle =
    targetKind === DatagraphItemKind.reply
      ? t.report.reportReply
      : t.report.reportThread;

  const modalDescription =
    targetKind === DatagraphItemKind.reply
      ? t.report.letModeratorsKnowReply
      : t.report.letModeratorsKnowThread;

  const submitLabel =
    targetKind === DatagraphItemKind.reply
      ? t.report.reportReply
      : t.report.reportThread;

  const successMessage =
    targetKind === DatagraphItemKind.reply
      ? t.report.replySubmitted
      : t.report.threadSubmitted;

  return (
    <ReportModal
      title={modalTitle}
      description={modalDescription}
      subject={
        <VStack alignItems="start" gap="2">
          <MemberBadge profile={author} size="sm" name="full-horizontal" />
          {headline && (
            <styled.span
              fontWeight="medium"
              maxW="64"
              whiteSpace="pre-wrap"
              wordBreak="break-word"
            >
              {headline}
            </styled.span>
          )}
          {body && (
            <styled.p
              fontSize="sm"
              color="fg.subtle"
              whiteSpace="pre-wrap"
              maxW="64"
              wordBreak="break-word"
            >
              {body}
            </styled.p>
          )}
        </VStack>
      }
      targetId={targetId}
      targetKind={targetKind}
      submitLabel={submitLabel}
      successMessage={successMessage}
      loadingMessage="Sending report..."
      {...disclosure}
    />
  );
}
