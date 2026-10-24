"use client";

import { ChangeEvent, useRef, useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import { Asset } from "@/api/openapi-schema";
import { useImageUpload } from "@/components/content/useImageUpload";
import { Button } from "@/components/ui/button";
import { IconButton } from "@/components/ui/icon-button";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import { MediaAddIcon } from "@/components/ui/icons/Media";
import { useTranslation } from "@/lib/i18n";
import { useMaxUploadSizeBytes } from "@/lib/settings/uploads";
import { Box, Grid, LStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

export const MAX_LISTING_IMAGES = 10;

type Props = {
  value: Asset[];
  onChange: (assets: Asset[]) => void;
  onUploadingChange?: (uploading: boolean) => void;
};

export function ListingImagesField({
  value,
  onChange,
  onUploadingChange,
}: Props) {
  const t = useTranslation();
  const inputRef = useRef<HTMLInputElement>(null);
  const { upload } = useImageUpload();
  const maxUploadSizeBytes = useMaxUploadSizeBytes();
  const [uploading, setUploading] = useState(false);

  const remaining = MAX_LISTING_IMAGES - value.length;

  async function handleFiles(e: ChangeEvent<HTMLInputElement>) {
    const files = Array.from(e.target.files ?? []).slice(0, remaining);
    e.target.value = "";
    if (files.length === 0) {
      return;
    }

    setUploading(true);
    onUploadingChange?.(true);

    let next = value;
    for (const file of files) {
      if (file.size > maxUploadSizeBytes) {
        toast.error(
          t.upload.tooLarge.replace(
            "{size}",
            String(Math.floor(maxUploadSizeBytes / 1024 / 1024)),
          ),
        );
        continue;
      }

      const asset = await handle(() => upload(file));
      if (asset) {
        next = [...next, asset];
        onChange(next);
      }
    }

    setUploading(false);
    onUploadingChange?.(false);
  }

  function handleRemove(id: string) {
    onChange(value.filter((a) => a.id !== id));
  }

  function handleMakeCover(id: string) {
    const target = value.find((a) => a.id === id);
    if (!target) return;
    onChange([target, ...value.filter((a) => a.id !== id)]);
  }

  return (
    <LStack gap="2" w="full">
      {value.length > 0 && (
        <Grid
          w="full"
          gap="2"
          gridTemplateColumns={{ base: "repeat(3, 1fr)", md: "repeat(5, 1fr)" }}
        >
          {value.map((a, i) => (
            <Box
              key={a.id}
              position="relative"
              borderRadius="md"
              overflow="hidden"
              borderWidth={i === 0 ? "medium" : "thin"}
              borderStyle="solid"
              borderColor={i === 0 ? "border.accent" : "border.subtle"}
              data-testid="listing-image"
            >
              <styled.img
                src={getAssetURL(a.path)}
                alt={a.filename}
                w="full"
                aspectRatio="1"
                objectFit="cover"
              />
              <Box position="absolute" top="1" right="1">
                <IconButton
                  type="button"
                  size="xs"
                  variant="subtle"
                  bg="bg.default"
                  aria-label={t.marketplace.form.removeImage}
                  title={t.marketplace.form.removeImage}
                  onClick={() => handleRemove(a.id)}
                >
                  <DeleteIcon />
                </IconButton>
              </Box>
              {i > 0 && (
                <Box position="absolute" bottom="1" left="1" right="1">
                  <Button
                    type="button"
                    size="xs"
                    variant="subtle"
                    bg="bg.default"
                    w="full"
                    onClick={() => handleMakeCover(a.id)}
                  >
                    {t.marketplace.form.makeCover}
                  </Button>
                </Box>
              )}
            </Box>
          ))}
        </Grid>
      )}

      <input
        ref={inputRef}
        type="file"
        accept="image/png,image/jpeg,image/webp,image/gif,image/avif,image/heic"
        multiple
        hidden
        onChange={handleFiles}
        data-testid="listing-image-input"
      />
      <Button
        type="button"
        size="sm"
        variant="outline"
        disabled={remaining <= 0}
        loading={uploading}
        onClick={() => inputRef.current?.click()}
      >
        <MediaAddIcon />
        {t.marketplace.form.addImage}
      </Button>
    </LStack>
  );
}
