"use client";

import { usePathname } from "next/navigation";

import { IconButton } from "@/components/ui/icon-button";
import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useTranslation } from "@/lib/i18n";
import { useAIFeature } from "@/lib/settings/ai-features";
import { css } from "@/styled-system/css";
import { HStack, LStack, styled } from "@/styled-system/jsx";
import { normalizeAssetPath } from "@/utils/asset";

import { plainText, useAttachmentSummary } from "./useAttachmentSummary";

const summaryContent = css({
  fontSize: "sm",
  "& p": { mt: "2" },
  "& p:first-child": { mt: "0" },
  "& ul": { listStyleType: "disc", pl: "5", mt: "1" },
  "& li": { mt: "1" },
});

const xidPrefix = /^[0-9a-v]{20}/;

export function attachmentIDFromRef(ref?: string | null): string | null {
  const m = normalizeAssetPath(ref).match(xidPrefix);
  return m ? m[0] : null;
}

export function isPDFAttachment(mimeType?: string, filename?: string) {
  if (mimeType === "application/pdf") return true;
  return /[.-]pdf$/i.test(filename ?? "");
}

function useThreadSlug() {
  const pathname = usePathname();
  const m = pathname?.match(/^\/t\/([^/]+)/);
  return m ? decodeURIComponent(m[1]!) : null;
}

type Props = {
  assetRef?: string | null;
  filename?: string;
  mimeType?: string;
  displayName: string;
};

export function AttachmentSummaryButton({
  assetRef,
  filename,
  mimeType,
  displayName,
}: Props) {
  const t = useTranslation();
  const enabled = useAIFeature("pdf_summary");
  const slug = useThreadSlug();
  const assetID = attachmentIDFromRef(assetRef);

  if (!enabled || !slug || !assetID || !isPDFAttachment(mimeType, filename)) {
    return null;
  }

  return (
    <AttachmentSummaryTrigger
      slug={slug}
      assetID={assetID}
      label={t.thread.pdfSummarise}
      displayName={displayName}
    />
  );
}

function AttachmentSummaryTrigger({
  slug,
  assetID,
  label,
  displayName,
}: {
  slug: string;
  assetID: string;
  label: string;
  displayName: string;
}) {
  const t = useTranslation();
  const { state, open, close } = useAttachmentSummary(slug, assetID);

  return (
    <styled.span
      display="inline-flex"
      onClick={(e) => e.stopPropagation()}
    >
      <IconButton
        type="button"
        size="xs"
        variant="ghost"
        aria-label={label}
        title={label}
        onClick={(e) => {
          e.preventDefault();
          e.stopPropagation();
          void open();
        }}
      >
        <IntelligenceIcon />
      </IconButton>

      <ModalDrawer
        isOpen={state.kind !== "idle"}
        onClose={close}
        title={`${t.thread.pdfSummaryTitle}: ${displayName}`}
      >
        <LStack gap="3">
          <SummaryBody state={state} />
          <styled.p fontSize="xs" color="fg.subtle">
            {t.thread.summaryDisclaimer}
          </styled.p>
        </LStack>
      </ModalDrawer>
    </styled.span>
  );
}

function SummaryBody({
  state,
}: {
  state: ReturnType<typeof useAttachmentSummary>["state"];
}) {
  const t = useTranslation();

  switch (state.kind) {
    case "idle":
      return null;

    case "loading":
      return (
        <HStack gap="2" color="fg.muted" fontSize="sm">
          <IntelligenceIcon width="4" height="4" />
          {t.thread.summarising}
        </HStack>
      );

    case "unavailable":
      return <styled.p fontSize="sm">{t.thread.pdfSummaryUnavailable}</styled.p>;

    case "streaming":
      return (
        <LStack gap="3">
          {state.text && (
            <styled.p
              fontSize="sm"
              whiteSpace="pre-line"
              data-testid="pdf-summary-live"
            >
              {plainText(state.text)}
            </styled.p>
          )}
          <Preview html={state.preview} hint={t.thread.pdfSummaryQuickHint} />
        </LStack>
      );

    case "done":
      return (
        <LStack gap="2">
          <div
            className={summaryContent}
            data-testid="pdf-summary"
            dangerouslySetInnerHTML={{ __html: state.summary }}
          />
          {state.quality === "quick" && (
            <styled.p fontSize="xs" color="fg.muted">
              {t.thread.pdfSummaryQuickLive}
            </styled.p>
          )}
        </LStack>
      );

    case "failed":
      return (
        <LStack gap="3">
          <styled.p fontSize="sm" color="fg.destructive">
            {t.thread.pdfSummaryFailed}
          </styled.p>
          {state.preview && <Preview html={state.preview} />}
        </LStack>
      );
  }
}

function Preview({ html, hint }: { html: string; hint?: string }) {
  const t = useTranslation();
  return (
    <LStack
      gap="2"
      p="3"
      w="full"
      borderRadius="md"
      borderWidth="thin"
      borderColor="border.subtle"
      bg="bg.muted"
    >
      <HStack gap="2" color="fg.muted" fontSize="sm" fontWeight="medium">
        {t.thread.pdfSummaryQuick}
      </HStack>
      <div
        className={summaryContent}
        data-testid="pdf-summary-preview"
        dangerouslySetInnerHTML={{ __html: html }}
      />
      {hint && (
        <styled.p fontSize="xs" color="fg.subtle">
          {hint}
        </styled.p>
      )}
    </LStack>
  );
}
