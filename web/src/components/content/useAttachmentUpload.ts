"use client";

import { useState } from "react";

import { handle } from "@/api/client";
import { assetUpload } from "@/api/openapi-client/assets";
import { Asset } from "@/api/openapi-schema";
import {
  MAX_ASSET_UPLOAD_BYTES,
  buildAttachmentMarkup,
} from "@/utils/asset";

import { isSupportedAsset } from "./useImageUpload";

export type UploadedAttachment = {
  asset: Asset;
  markup: string;
};

/**
 * Shared attachment upload for the simple composers (reply box, quick share)
 * that append markup to a form body rather than driving editor nodes.
 *
 * Both used to inline their own near-identical copy of this, each swallowing
 * upload failures in a bare catch — so a rejected file silently produced a post
 * with no attachment. Failures now surface as a toast and `isUploading` gives
 * callers something to disable their attach control with, which is what stops a
 * second click queueing a duplicate upload of the same file.
 */
export function useAttachmentUpload() {
  const [isUploading, setUploading] = useState(false);

  async function upload(files: File[]): Promise<UploadedAttachment[]> {
    if (files.length === 0) {
      return [];
    }

    setUploading(true);

    const uploaded: UploadedAttachment[] = [];

    try {
      for (const file of files) {
        await handle(async () => {
          if (file.size > MAX_ASSET_UPLOAD_BYTES) {
            throw new Error(
              `${file.name} is larger than the ${Math.floor(MAX_ASSET_UPLOAD_BYTES / 1024 / 1024)}MB upload limit.`,
            );
          }

          if (!isSupportedAsset(file.type)) {
            throw new Error(`${file.name} is not a supported file type.`);
          }

          const asset = await assetUpload(file, { filename: file.name });

          uploaded.push({
            asset: { ...asset, filename: file.name },
            markup: buildAttachmentMarkup(asset.path, file),
          });
        });
      }
    } finally {
      setUploading(false);
    }

    return uploaded;
  }

  return { isUploading, upload };
}
