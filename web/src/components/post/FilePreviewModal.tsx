"use client";

import React, { useEffect, useState } from "react";
import { Portal } from "@ark-ui/react";
import { X, Download, FileText } from "lucide-react";
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
  const previewType = detectPreviewType(mimeType, displayName);
  const [textContent, setTextContent] = useState<string | null>(null);
  const [showOcr, setShowOcr] = useState(false);
  const [mounted, setMounted] = useState(false);

  useEffect(() => {
    setMounted(true);
  }, []);

  // Load text files
  useEffect(() => {
    if (previewType === "text" && url) {
      fetch(url, { credentials: "include" })
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

  if (!mounted) return null;

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
          width: "90vw",
          height: "90vh",
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
                title={showOcr ? "Bild anzeigen" : "Text im Bild anzeigen"}
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
                <span>{showOcr ? "Bild anzeigen" : "Text im Bild"}</span>
              </styled.button>
            )}

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
          {showOcr && ocrText ? (
            <styled.div
              w="full"
              h="full"
              overflow="auto"
              p="6"
              bgColor="bg.subtle"
            >
              <styled.h4 fontSize="xs" fontWeight="bold" textTransform="uppercase" color="fg.muted" mb="3">
                Extrahierter Text (OCR)
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
            <>
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
            </>
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
