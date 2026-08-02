"use client";

import React, { useEffect, useState } from "react";
import { Portal } from "@ark-ui/react";
import { X, Download } from "lucide-react";
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
  if (mime === "text/plain" || mime === "text/csv") return true;

  // Fallback: check extension in filename
  if (/\.(pdf)$/i.test(name)) return true;
  if (/\.(txt|csv)$/i.test(name)) return true;

  return false;
}

type PreviewType = "pdf" | "image" | "text" | null;

function detectPreviewType(
  mimeType?: string,
  filename?: string
): PreviewType {
  const mime = (mimeType ?? "").toLowerCase();
  const name = (filename ?? "").toLowerCase();

  if (mime.startsWith("image/")) return "image";
  if (mime === "application/pdf" || /\.pdf$/i.test(name)) return "pdf";
  if (
    mime === "text/plain" ||
    mime === "text/csv" ||
    /\.(txt|csv)$/i.test(name)
  )
    return "text";

  return null;
}

interface FilePreviewModalProps {
  url: string;
  displayName: string;
  mimeType?: string;
  onClose: () => void;
  onDownload: () => void;
}

export function FilePreviewModal({
  url,
  displayName,
  mimeType,
  onClose,
  onDownload,
}: FilePreviewModalProps) {
  const previewType = detectPreviewType(mimeType, displayName);
  const [textContent, setTextContent] = useState<string | null>(null);

  // Load text files
  useEffect(() => {
    if (previewType === "text" && url) {
      fetch(url)
        .then((r) => r.text())
        .then(setTextContent)
        .catch(() => setTextContent("(Datei konnte nicht geladen werden)"));
    }
  }, [previewType, url]);

  // Close on Escape
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [onClose]);

  return (
    <Portal>
      {/* Backdrop */}
      <styled.div
        position="fixed"
        inset="0"
        zIndex="99998"
        backgroundColor="rgba(0, 0, 0, 0.5)"
        backdropFilter="blur(6px)"
        onClick={onClose}
        style={{ cursor: "default" }}
      />

      {/* Modal panel */}
      <styled.div
        position="fixed"
        zIndex="99999"
        top="50%"
        left="50%"
        style={{ transform: "translate(-50%, -50%)" }}
        display="flex"
        flexDirection="column"
        width="90vw"
        height="90vh"
        borderRadius="xl"
        bgColor="bg.default"
        boxShadow="2xl"
        overflow="hidden"
      >
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
            {/* Download button */}
            <styled.button
              type="button"
              onClick={onDownload}
              title="Herunterladen"
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
              title="Schließen"
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
          {previewType === "pdf" && (
            <iframe
              src={url}
              title={displayName}
              style={{ width: "100%", height: "100%", border: "none" }}
            />
          )}

          {previewType === "image" && (
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
          )}

          {previewType === "text" && (
            <styled.div
              w="full"
              h="full"
              overflow="auto"
              p="4"
              bgColor="bg.subtle"
            >
              <styled.pre
                fontSize="sm"
                fontFamily="mono"
                color="fg.default"
                whiteSpace="pre-wrap"
                wordBreak="break-all"
              >
                {textContent ?? "Wird geladen…"}
              </styled.pre>
            </styled.div>
          )}

          {!previewType && (
            <styled.div
              display="flex"
              alignItems="center"
              justifyContent="center"
              w="full"
              h="full"
              color="fg.muted"
              fontSize="sm"
            >
              Keine Vorschau verfügbar
            </styled.div>
          )}
        </styled.div>
      </styled.div>
    </Portal>
  );
}
