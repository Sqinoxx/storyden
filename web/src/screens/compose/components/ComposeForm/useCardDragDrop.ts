"use client";

import { useRef, useState } from "react";

import { Asset } from "@/api/openapi-schema";

import { handle } from "@/api/client";
import {
  hasAssetFile,
  isSupportedAsset,
  useImageUpload,
} from "@/components/content/useImageUpload";
import { useMaxUploadSizeBytes } from "@/lib/settings/uploads";

const ERROR_UNSUPPORTED_FILE_TYPE = "File type not supported";

// Drop target for the whole compose card (title, body and footer). Distinct
// from the rich/markdown editors' own drag handling, which additionally
// embeds images inline at the cursor when dropped directly onto the text —
// this one always just attaches, matching the "Datei anhängen" button. Those
// inner handlers stop propagation on drag events precisely so a drop over the
// editor doesn't also get picked up here and uploaded twice.
export function useCardDragDrop(onAttach: (asset: Asset) => Promise<void>) {
  const { upload } = useImageUpload();
  const maxUploadSizeBytes = useMaxUploadSizeBytes();

  const [isDragging, setIsDragging] = useState(false);
  const [isDragError, setIsDragError] = useState(false);
  const [dragErrorMessage, setDragErrorMessage] = useState("");
  const [dragFileCount, setDragFileCount] = useState(0);
  const dragCounterRef = useRef(0);

  function getDragOverlayMessage() {
    if (isDragError) {
      return dragErrorMessage;
    }
    return dragFileCount === 1
      ? "Drop 1 file to upload"
      : `Drop ${dragFileCount} files to upload`;
  }

  function handleDragOver(e: React.DragEvent) {
    e.preventDefault();
  }

  function handleDragEnter(e: React.DragEvent) {
    dragCounterRef.current += 1;

    const items = Array.from(e.dataTransfer.items);
    const hasFile = items.some((item) => item.kind === "file");

    if (!hasFile) {
      return;
    }

    e.preventDefault();
    setIsDragging(true);

    const hasAsset = hasAssetFile(e.dataTransfer.items);
    const assetCount = items.filter((item) =>
      isSupportedAsset(item.type),
    ).length;

    if (!hasAsset) {
      setIsDragError(true);
      setDragErrorMessage(ERROR_UNSUPPORTED_FILE_TYPE);
    } else {
      setIsDragError(false);
      setDragErrorMessage("");
    }

    setDragFileCount(assetCount);
  }

  function handleDragLeave() {
    dragCounterRef.current = Math.max(0, dragCounterRef.current - 1);
    if (dragCounterRef.current === 0) {
      setIsDragging(false);
      setIsDragError(false);
      setDragErrorMessage("");
      setDragFileCount(0);
    }
  }

  async function handleDrop(e: React.DragEvent) {
    e.preventDefault();

    dragCounterRef.current = 0;
    setIsDragging(false);
    setIsDragError(false);
    setDragErrorMessage("");
    setDragFileCount(0);

    const files = Array.from(e.dataTransfer.files).filter((file) =>
      isSupportedAsset(file.type),
    );

    for (const file of files) {
      await handle(async () => {
        if (file.size > maxUploadSizeBytes) {
          throw new Error(
            `File is larger than the ${Math.floor(maxUploadSizeBytes / 1024 / 1024)}MB upload limit.`,
          );
        }

        const asset = await upload(file, { filename: file.name });
        await onAttach({ ...asset, filename: file.name });
      });
    }
  }

  return {
    isDragging,
    isDragError,
    getDragOverlayMessage,
    handlers: {
      handleDragOver,
      handleDragEnter,
      handleDragLeave,
      handleDrop,
    },
  };
}
