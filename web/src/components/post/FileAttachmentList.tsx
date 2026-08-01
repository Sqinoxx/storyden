"use client";

import React from "react";
import { Download } from "lucide-react";

import { Asset } from "@/api/openapi-schema";
import { getAssetURL, getCleanFilename } from "@/utils/asset";
import { styled } from "@/styled-system/jsx";

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

  function handleDownload(e: React.MouseEvent) {
    e.preventDefault();
    e.stopPropagation();
    if (url) {
      downloadAsset(url, displayName);
    }
  }

  return (
    <styled.a
      href={url}
      download={displayName}
      title={displayName}
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
      textDecoration="none"
      fontSize="sm"
      fontWeight="medium"
      maxWidth="xs"
      overflow="hidden"
      position="relative"
      cursor="pointer"
      _hover={{ bgColor: "bg.muted" }}
      onClick={handleDownload}
      style={{ zIndex: 10, transition: "background 0.15s" }}
    >
      <styled.span display="flex" alignItems="center" flexShrink="0" fontSize="sm">
        📄
      </styled.span>
      <styled.span
        overflow="hidden"
        textOverflow="ellipsis"
        whiteSpace="nowrap"
        flex="1"
        minWidth="0"
      >
        {displayName}
      </styled.span>
      <styled.span display="flex" alignItems="center" color="fg.muted" flexShrink="0" ml="1">
        <Download size={14} />
      </styled.span>
    </styled.a>
  );
}

/** Renders a list of document asset download badges. */
export function FileAttachmentList({
  assets,
  body,
}: {
  assets: Asset[];
  body?: string;
}) {
  let documentAssets = assets.filter(isDocumentAsset);

  if (body) {
    const bodyLower = body.toLowerCase();
    documentAssets = documentAssets.filter((asset) => {
      if (!asset.path && !asset.filename) return true;
      const pathKey = asset.path ? asset.path.replace(/^.*\/api\/assets\//, "") : "";
      const nameKey = asset.filename ? asset.filename.toLowerCase() : "";

      if (pathKey && bodyLower.includes(pathKey.toLowerCase())) return false;
      if (asset.path && bodyLower.includes(asset.path.toLowerCase())) return false;
      if (nameKey && bodyLower.includes(nameKey)) return false;
      return true;
    });
  }

  // Filter out redundant '-untitled' fallback assets if a properly named asset exists
  if (documentAssets.length > 1) {
    const hasNamedAsset = documentAssets.some(
      (a) => a.filename && !a.filename.toLowerCase().endsWith("-untitled") && a.filename.toLowerCase() !== "untitled"
    );
    if (hasNamedAsset) {
      documentAssets = documentAssets.filter(
        (a) => a.filename && !a.filename.toLowerCase().endsWith("-untitled") && a.filename.toLowerCase() !== "untitled"
      );
    }
  }

  if (documentAssets.length === 0) return null;

  return (
    <styled.div
      display="flex"
      flexWrap="wrap"
      gap="2"
      mt="2"
    >
      {documentAssets.map((asset) => (
        <FileAttachmentBadge key={asset.id || asset.filename} asset={asset} />
      ))}
    </styled.div>
  );
}
