"use client";

import React, { useState, useCallback } from "react";
import { Download, Eye } from "lucide-react";

import { Asset } from "@/api/openapi-schema";
import {
  getAssetURL,
  getCleanFilename,
  normalizeAssetPath,
  normalizeFilename,
} from "@/utils/asset";
import { styled } from "@/styled-system/jsx";
import { FilePreviewModal, isPreviewableAsset } from "./FilePreviewModal";
import { extractDocumentAssetsFromThread } from "./ThreadCard";

/** Returns true if the asset is a document (not an image). */
export function isDocumentAsset(asset: Asset) {
  if (!asset) return false;
  if (asset.mime_type && asset.mime_type.startsWith("image/")) {
    return false;
  }
  return true;
}

/** Utility function to trigger a clean file download without opening blank tabs. */
async function downloadAsset(url: string, filename: string) {
  try {
    const res = await fetch(url);
    if (!res.ok) throw new Error("Download failed");
    const blob = await res.blob();
    const blobUrl = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = blobUrl;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(blobUrl);
  } catch {
    const link = document.createElement("a");
    link.href = url;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
  }
}

/** A compact, styled badge for a single file attachment. */
export function FileAttachmentBadge({ asset }: { asset: Asset }) {
  const url = getAssetURL(asset.path);
  const displayName = getCleanFilename(asset.filename);
  const isImage = asset.mime_type?.startsWith("image/");
  const canPreview = isPreviewableAsset(asset.mime_type, asset.filename);

  const [previewOpen, setPreviewOpen] = useState(false);

  const handleDownload = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (url) {
        downloadAsset(url, displayName);
      }
    },
    [url, displayName]
  );

  const handlePreview = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      setPreviewOpen(true);
    },
    []
  );

  const handleCardClick = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      e.stopPropagation();
      if (canPreview) {
        setPreviewOpen(true);
      } else if (url) {
        downloadAsset(url, displayName);
      }
    },
    [canPreview, url, displayName]
  );

  return (
    <>
      <styled.div
        display="inline-flex"
        alignItems="center"
        gap="2"
        px="3"
        py="2"
        borderRadius="md"
        borderWidth="thin"
        borderColor="border.default"
        bgColor="bg.subtle"
        color="fg.default"
        fontSize="sm"
        fontWeight="medium"
        maxWidth="xs"
        overflow="hidden"
        position="relative"
        title={canPreview ? `${displayName} – Vorschau öffnen` : `${displayName} – Herunterladen`}
        onClick={handleCardClick}
        style={{ zIndex: 10, cursor: "pointer" }}
        _hover={{ bgColor: "bg.muted" }}
      >
        {/* File type icon */}
        {isImage ? (
          <styled.img
            src={url ?? ""}
            alt={displayName}
            style={{
              width: "20px",
              height: "20px",
              objectFit: "cover",
              borderRadius: "3px",
              flexShrink: 0,
            }}
          />
        ) : (
          <styled.span display="flex" alignItems="center" flexShrink="0" fontSize="sm">
            📄
          </styled.span>
        )}

        {/* Filename */}
        <styled.span
          overflow="hidden"
          textOverflow="ellipsis"
          whiteSpace="nowrap"
          flex="1"
          minWidth="0"
        >
          {displayName}
        </styled.span>

        {/* Action buttons */}
        <styled.span display="flex" alignItems="center" gap="1.5" flexShrink="0" ml="1">
          {/* Preview eye icon – only for supported formats */}
          {canPreview && (
            <styled.button
              type="button"
              onClick={handlePreview}
              title="Vorschau"
              display="inline-flex"
              alignItems="center"
              justifyContent="center"
              color="fg.muted"
              _hover={{ color: "fg.default" }}
              style={{
                cursor: "pointer",
                background: "transparent",
                border: "none",
                padding: "2px",
              }}
            >
              <Eye size={14} />
            </styled.button>
          )}

          {/* Download icon */}
          <styled.button
            type="button"
            onClick={handleDownload}
            title="Herunterladen"
            display="inline-flex"
            alignItems="center"
            justifyContent="center"
            color="fg.muted"
            _hover={{ color: "fg.default" }}
            style={{
              cursor: "pointer",
              background: "transparent",
              border: "none",
              padding: "2px",
            }}
          >
            <Download size={14} />
          </styled.button>
        </styled.span>
      </styled.div>

      {/* Preview modal */}
      {previewOpen && url && (
        <FilePreviewModal
          url={url}
          displayName={displayName}
          mimeType={asset.mime_type}
          onClose={() => setPreviewOpen(false)}
          onDownload={() => downloadAsset(url, displayName)}
        />
      )}
    </>
  );
}

/** Renders a list of all asset download badges (images + documents). */
export function FileAttachmentList({
  assets,
  body,
}: {
  assets: Asset[];
  body?: string;
}) {
  const visibleAssets = extractDocumentAssetsFromThread({ assets, body });

  // Filter out assets that are ALREADY embedded as links in the body HTML,
  // since ContentComposerRich renders embedded file attachment links inline.
  const unembeddedAssets = visibleAssets.filter((asset) => {
    if (!body) return true;
    const normP = normalizeAssetPath(asset.path ?? asset.id);
    const normN = normalizeFilename(asset.filename);
    const cleanN = getCleanFilename(asset.filename).toLowerCase();

    try {
      const anchorRegex = /<a\b([^>]*)>(.*?)<\/a>/gi;
      let match: RegExpExecArray | null;
      while ((match = anchorRegex.exec(body)) !== null) {
        const attrString = match[1] ?? "";
        const hrefMatch = /href=["']([^"']+)["']/i.exec(attrString);
        const href = hrefMatch ? hrefMatch[1] : null;

        if (href) {
          const filenameMatch = /data-filename=["']([^"']+)["']/i.exec(attrString);
          const fnAttr = filenameMatch ? filenameMatch[1] : null;
          const innerText = (match[2] ?? "").replace(/<[^>]+>/g, "").trim();
          const fn = fnAttr || innerText || href.split("/").pop() || "";

          const elNormP = normalizeAssetPath(href);
          const elNormN = normalizeFilename(fn);
          const elCleanN = getCleanFilename(fn).toLowerCase();

          const pathMatch =
            normP &&
            elNormP &&
            (normP === elNormP ||
              (asset.id && href.toLowerCase().includes(asset.id.toLowerCase())));
          const nameMatch =
            (normN && elNormN && normN === elNormN) ||
            (cleanN && elCleanN && cleanN === elCleanN);

          if (pathMatch || nameMatch) {
            return false;
          }
        }
      }
    } catch {
      // ignore regex parsing issues
    }
    return true;
  });

  if (unembeddedAssets.length === 0) return null;

  return (
    <styled.div
      display="flex"
      flexWrap="wrap"
      gap="2"
      mt="2"
    >
      {unembeddedAssets.map((asset) => (
        <FileAttachmentBadge key={asset.id || asset.path || asset.filename} asset={asset} />
      ))}
    </styled.div>
  );
}
