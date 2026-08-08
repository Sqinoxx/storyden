"use client";

import React, { useState, useCallback } from "react";
import { Download, Eye, Trash2 } from "lucide-react";

import { Asset } from "@/api/openapi-schema";
import {
  downloadAsset,
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

/** A compact, styled badge for a single file attachment. */
export function FileAttachmentBadge({
  asset,
  onRemove,
}: {
  asset: Asset;
  onRemove?: () => void;
}) {
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

          {/* Delete icon – when onRemove is provided (e.g. during editing) */}
          {onRemove && (
            <styled.button
              type="button"
              onClick={(e: React.MouseEvent) => {
                e.preventDefault();
                e.stopPropagation();
                onRemove();
              }}
              title="Datei löschen"
              display="inline-flex"
              alignItems="center"
              justifyContent="center"
              color="fg.muted"
              _hover={{ color: "red.5" }}
              style={{
                cursor: "pointer",
                background: "transparent",
                border: "none",
                padding: "2px",
              }}
            >
              <Trash2 size={14} />
            </styled.button>
          )}
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
  onRemoveAsset,
}: {
  assets: Asset[];
  body?: string;
  onRemoveAsset?: (asset: Asset) => void;
}) {
  const visibleAssets = extractDocumentAssetsFromThread({ assets, body });

  // When not editing, filter out assets that are ALREADY embedded as links in the body HTML,
  // since ContentComposerRich renders embedded file attachment links inline.
  // When editing (onRemoveAsset provided), show all visible document assets so the user can easily manage/delete them.
  const unembeddedAssets = visibleAssets.filter((asset) => {
    if (!body) return true;
    const normP = normalizeAssetPath(asset.path ?? asset.id);
    const normN = normalizeFilename(asset.filename);
    const cleanN = getCleanFilename(asset.filename).toLowerCase();

    try {
      const parsed = new DOMParser().parseFromString(body, "text/html");
      const elements = parsed.querySelectorAll(
        "a, span[data-type='file-attachment'], div[data-type='file-attachment']"
      );
      
      for (const el of elements) {
        const href = el.getAttribute("href") || el.getAttribute("src") || "";
        if (!href) continue;

        const fnAttr = el.getAttribute("data-filename") || el.getAttribute("download") || "";
        const innerText = el.textContent?.trim() || "";
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
    } catch {
      // ignore DOMParser parsing issues
    }
    return true;
  });

  const listToRender = unembeddedAssets;

  if (listToRender.length === 0) return null;

  return (
    <styled.div
      display="flex"
      flexWrap="wrap"
      gap="2"
      mt="2"
    >
      {listToRender.map((asset) => (
        <FileAttachmentBadge
          key={asset.id || asset.path || asset.filename}
          asset={asset}
          onRemove={onRemoveAsset ? () => onRemoveAsset(asset) : undefined}
        />
      ))}
    </styled.div>
  );
}

