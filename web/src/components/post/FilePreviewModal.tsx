"use client";

import React, { PropsWithChildren, useEffect, useState } from "react";
import { Portal } from "@ark-ui/react";
import { X, Download, FileText } from "lucide-react";

import { recheckSession } from "@/api/client";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { styled } from "@/styled-system/jsx";

/** Mime types / extensions that can be previewed inline in the browser. */
export function isPreviewableAsset(
  mimeType?: string,
  filename?: string
): boolean {
  if (!mimeType && !filename) return false;

  const mime = (mimeType ?? "").toLowerCase();
  const name = (filename ?? "").toLowerCase();

  if (mime.startsWith("image/")) return true;
  if (mime === "application/pdf") return true;
  if (mime.startsWith("text/")) return true;
  if (mime === "application/json") return true;

  // Fallback: check extension in filename
  if (/\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)$/i.test(name)) return true;
  if (/\.(pdf)$/i.test(name)) return true;
  if (/\.(txt|csv|json|md|log|js|ts|jsx|tsx|html|css|xml|yaml|yml)$/i.test(name))
    return true;

  return false;
}

type PreviewType = "pdf" | "image" | "text" | null;

function detectPreviewType(
  mimeType?: string,
  filename?: string
): PreviewType {
  const mime = (mimeType ?? "").toLowerCase();
  const name = (filename ?? "").toLowerCase();

  if (
    mime.startsWith("image/") ||
    /\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)$/i.test(name)
  )
    return "image";
  if (mime === "application/pdf" || /\.pdf$/i.test(name)) return "pdf";
  if (
    mime.startsWith("text/") ||
    mime === "application/json" ||
    /\.(txt|csv|json|md|log|js|ts|jsx|tsx|html|css|xml|yaml|yml)$/i.test(name)
  )
    return "text";

  return null;
}

type AssetAccess = "checking" | "ok" | "unauthenticated" | "failed";

type AssetProbe = { access: AssetAccess; text?: string };

/**
 * The PDF and image viewers render whatever the server returns, so an expired
 * session would show the API's raw 401 body. Probing first lets that case be
 * shown as a sign-in prompt instead.
 */
async function probeAsset(
  url: string,
  readText: boolean,
  signal: AbortSignal,
): Promise<AssetProbe> {
  const init: RequestInit = readText
    ? { credentials: "include", signal }
    : {
        credentials: "include",
        signal,
        cache: "no-store",
        headers: { Range: "bytes=0-0" },
      };

  try {
    const response = await fetch(url, init);

    if (response.status === 401) return { access: "unauthenticated" };
    if (!response.ok) return { access: "failed" };
    if (readText) return { access: "ok", text: await response.text() };

    void response.body?.cancel().catch(() => {});
    return { access: "ok" };
  } catch {
    // Links to other sites fail CORS here even though a viewer can load them.
    return { access: readText ? "failed" : "ok" };
  }
}

interface FilePreviewModalProps {
  url: string;
  displayName: string;
  mimeType?: string;
  ocrText?: string;
  onClose: () => void;
  onDownload: () => void;
}

export function FilePreviewModal({
  url,
  displayName,
  mimeType,
  ocrText,
  onClose,
  onDownload,
}: FilePreviewModalProps) {
  const t = useTranslation();
  const previewType = detectPreviewType(mimeType, displayName);
  const [probe, setProbe] = useState<AssetProbe>({ access: "checking" });
  const [showOcr, setShowOcr] = useState(false);
  const [mounted, setMounted] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  useEffect(() => {
    const mql = window.matchMedia("(max-width: 640px)");
    const update = () => setIsMobile(mql.matches);
    update();
    mql.addEventListener("change", update);
    return () => mql.removeEventListener("change", update);
  }, []);

  useEffect(() => {
    if (!previewType || !url) return;

    const controller = new AbortController();
    setProbe({ access: "checking" });

    probeAsset(url, previewType === "text", controller.signal).then(
      (result) => {
        if (controller.signal.aborted) return;
        if (result.access === "unauthenticated") recheckSession();
        setProbe(result);
      },
    );

    return () => controller.abort();
  }, [previewType, url]);

  // Close on Escape
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onClose]);

  if (!mounted) return null;

  const isMobilePdf = isMobile && previewType === "pdf";

  return (
    <Portal>
      {/* Backdrop */}
      <styled.div
        position="fixed"
        onClick={onClose}
        style={{
          top: "-20px",
          left: "-20px",
          right: "-20px",
          bottom: "-20px",
          zIndex: 99998,
          backgroundColor: "rgba(0, 0, 0, 0.5)",
          backdropFilter: "blur(6px)",
          WebkitBackdropFilter: "blur(6px)",
          transform: "translateZ(0)",
          willChange: "backdrop-filter, opacity",
          cursor: "default",
          animation: "previewFadeIn 0.15s ease-out forwards",
        }}
      />

      {/* Modal panel */}
      <styled.div
        position="fixed"
        display="flex"
        flexDirection="column"
        borderRadius="xl"
        bgColor="bg.default"
        boxShadow="2xl"
        overflow="hidden"
        style={{
          zIndex: 99999,
          top: "50%",
          left: "50%",
          transform: "translate(-50%, -50%) translateZ(0)",
          width: isMobilePdf ? "82vw" : "90vw",
          height: isMobilePdf ? "72vh" : "90vh",
          maxHeight:
            "calc(100dvh - env(safe-area-inset-top) - env(safe-area-inset-bottom) - var(--spacing-8))",
          animation: "previewZoomIn 0.15s cubic-bezier(0.16, 1, 0.3, 1) forwards",
        }}
      >
        <style jsx global>{`
          @keyframes previewFadeIn {
            from { opacity: 0; }
            to { opacity: 1; }
          }
          @keyframes previewZoomIn {
            from { opacity: 0; transform: translate(-50%, -48%) scale(0.97) translateZ(0); }
            to { opacity: 1; transform: translate(-50%, -50%) scale(1) translateZ(0); }
          }
        `}</style>
        {/* Header */}
        <styled.div
          display="flex"
          alignItems="center"
          justifyContent="space-between"
          px="4"
          py="3"
          borderBottomWidth="thin"
          borderColor="border.default"
          gap="3"
          flexShrink="0"
        >
          <styled.span
            fontSize="sm"
            fontWeight="medium"
            overflow="hidden"
            textOverflow="ellipsis"
            whiteSpace="nowrap"
            flex="1"
            minWidth="0"
            color="fg.default"
          >
            {displayName}
          </styled.span>

          <styled.div display="flex" alignItems="center" gap="2" flexShrink="0">
            {ocrText && (
              <styled.button
                type="button"
                onClick={() => setShowOcr((prev) => !prev)}
                title={showOcr ? t.filePreview.showImage : t.filePreview.showImageText}
                display="inline-flex"
                alignItems="center"
                gap="1.5"
                px="2.5"
                py="1"
                borderRadius="md"
                fontSize="xs"
                fontWeight="medium"
                bgColor={showOcr ? "bg.muted" : "bg.subtle"}
                color={showOcr ? "fg.default" : "fg.muted"}
                borderWidth="thin"
                borderColor="border.default"
                _hover={{ bgColor: "bg.muted", color: "fg.default" }}
                style={{ transition: "all 0.15s ease" }}
              >
                <FileText size={14} />
                <span>{showOcr ? t.filePreview.showImage : t.filePreview.imageText}</span>
              </styled.button>
            )}

            {/* Download button */}
            <styled.button
              type="button"
              onClick={onDownload}
              title={t.filePreview.download}
              display="inline-flex"
              alignItems="center"
              justifyContent="center"
              w="8"
              h="8"
              borderRadius="md"
              color="fg.muted"
              _hover={{ bgColor: "bg.muted", color: "fg.default" }}
              style={{ transition: "background 0.15s, color 0.15s" }}
            >
              <Download size={16} />
            </styled.button>

            {/* Close button */}
            <styled.button
              type="button"
              onClick={onClose}
              title={t.filePreview.close}
              display="inline-flex"
              alignItems="center"
              justifyContent="center"
              w="8"
              h="8"
              borderRadius="md"
              color="fg.muted"
              _hover={{ bgColor: "bg.muted", color: "fg.default" }}
              style={{ transition: "background 0.15s, color 0.15s" }}
            >
              <X size={16} />
            </styled.button>
          </styled.div>
        </styled.div>

        {/* Preview area */}
        <styled.div flex="1" overflow="hidden" position="relative">
          {showOcr && ocrText ? (
            <styled.div
              w="full"
              h="full"
              overflow="auto"
              p="6"
              bgColor="bg.subtle"
            >
              <styled.h4 fontSize="xs" fontWeight="bold" textTransform="uppercase" color="fg.muted" mb="3">
                {t.filePreview.ocrHeading}
              </styled.h4>
              <styled.pre
                fontSize="sm"
                fontFamily="mono"
                color="fg.default"
                whiteSpace="pre-wrap"
                wordBreak="break-word"
                p="4"
                borderRadius="lg"
                bgColor="bg.default"
                borderWidth="thin"
                borderColor="border.subtle"
              >
                {ocrText}
              </styled.pre>
            </styled.div>
          ) : (
            <PreviewBody
              previewType={previewType}
              probe={probe}
              url={url}
              displayName={displayName}
            />
          )}
        </styled.div>
      </styled.div>
    </Portal>
  );
}

function PreviewBody({
  previewType,
  probe,
  url,
  displayName,
}: {
  previewType: PreviewType;
  probe: AssetProbe;
  url: string;
  displayName: string;
}) {
  const t = useTranslation();

  if (!previewType) {
    return <PreviewNotice>{t.filePreview.noPreview}</PreviewNotice>;
  }

  switch (probe.access) {
    case "checking":
      return <PreviewNotice>{t.filePreview.loading}</PreviewNotice>;

    case "unauthenticated":
      return (
        <PreviewNotice>
          <span>{t.filePreview.sessionExpired}</span>
          <LinkButton href="/login" size="sm">
            {t.filePreview.signIn}
          </LinkButton>
        </PreviewNotice>
      );

    case "failed":
      return <PreviewNotice>{t.filePreview.loadFailed}</PreviewNotice>;
  }

  switch (previewType) {
    case "pdf":
      return (
        <iframe
          src={url}
          title={displayName}
          style={{ width: "100%", height: "100%", border: "none" }}
        />
      );

    case "image":
      return (
        <styled.div
          display="flex"
          alignItems="center"
          justifyContent="center"
          w="full"
          h="full"
          p="4"
          overflow="auto"
          bgColor="bg.subtle"
        >
          <styled.img
            src={url}
            alt={displayName}
            style={{
              maxWidth: "100%",
              maxHeight: "100%",
              objectFit: "contain",
              borderRadius: "8px",
            }}
          />
        </styled.div>
      );

    case "text":
      return (
        <styled.div w="full" h="full" overflow="auto" p="4" bgColor="bg.subtle">
          <styled.pre
            fontSize="sm"
            fontFamily="mono"
            color="fg.default"
            whiteSpace="pre-wrap"
            wordBreak="break-all"
          >
            {probe.text}
          </styled.pre>
        </styled.div>
      );
  }
}

function PreviewNotice({ children }: PropsWithChildren) {
  return (
    <styled.div
      display="flex"
      flexDirection="column"
      alignItems="center"
      justifyContent="center"
      gap="3"
      w="full"
      h="full"
      p="6"
      color="fg.muted"
      fontSize="sm"
      textAlign="center"
    >
      {children}
    </styled.div>
  );
}
