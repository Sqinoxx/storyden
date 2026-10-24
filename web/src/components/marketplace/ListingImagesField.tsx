"use client";

import {
  DndContext,
  DragEndEvent,
  KeyboardSensor,
  MouseSensor,
  TouchSensor,
  closestCenter,
  useSensor,
  useSensors,
} from "@dnd-kit/core";
import {
  SortableContext,
  arrayMove,
  rectSortingStrategy,
  sortableKeyboardCoordinates,
  useSortable,
} from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
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

  const sensors = useSensors(
    useSensor(MouseSensor, { activationConstraint: { distance: 5 } }),
    useSensor(TouchSensor, {
      activationConstraint: { delay: 200, tolerance: 5 },
    }),
    useSensor(KeyboardSensor, {
      coordinateGetter: sortableKeyboardCoordinates,
    }),
  );

  function handleDragEnd({ active, over }: DragEndEvent) {
    if (!over || active.id === over.id) return;
    const from = value.findIndex((a) => a.id === active.id);
    const to = value.findIndex((a) => a.id === over.id);
    if (from < 0 || to < 0) return;
    onChange(arrayMove(value, from, to));
  }

  return (
    <LStack gap="2" w="full">
      {value.length > 0 && (
        <DndContext
          sensors={sensors}
          collisionDetection={closestCenter}
          onDragEnd={handleDragEnd}
        >
          <SortableContext
            items={value.map((a) => a.id)}
            strategy={rectSortingStrategy}
          >
            <Grid
              w="full"
              gap="2"
              gridTemplateColumns={{
                base: "repeat(3, 1fr)",
                md: "repeat(5, 1fr)",
              }}
            >
              {value.map((a, i) => (
                <SortableImage
                  key={a.id}
                  asset={a}
                  isCover={i === 0}
                  onRemove={handleRemove}
                  onMakeCover={handleMakeCover}
                />
              ))}
            </Grid>
          </SortableContext>
        </DndContext>
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

type SortableImageProps = {
  asset: Asset;
  isCover: boolean;
  onRemove: (id: string) => void;
  onMakeCover: (id: string) => void;
};

function SortableImage({
  asset: a,
  isCover,
  onRemove,
  onMakeCover,
}: SortableImageProps) {
  const t = useTranslation();
  const {
    attributes,
    listeners,
    setNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: a.id });

  return (
    <Box
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform),
        transition,
      }}
      position="relative"
      borderRadius="md"
      overflow="hidden"
      borderWidth={isCover ? "medium" : "thin"}
      borderStyle="solid"
      borderColor={isCover ? "border.accent" : "border.subtle"}
      cursor={isDragging ? "grabbing" : "grab"}
      opacity={isDragging ? "7" : "full"}
      zIndex={isDragging ? "docked" : undefined}
      touchAction="manipulation"
      data-testid="listing-image"
      {...attributes}
      {...listeners}
    >
      <styled.img
        src={getAssetURL(a.path)}
        alt={a.filename}
        w="full"
        aspectRatio="1"
        objectFit="cover"
        draggable={false}
        userSelect="none"
      />
      <Box position="absolute" top="1" right="1">
        <IconButton
          type="button"
          size="xs"
          variant="subtle"
          bg="bg.default"
          aria-label={t.marketplace.form.removeImage}
          title={t.marketplace.form.removeImage}
          onClick={() => onRemove(a.id)}
        >
          <DeleteIcon />
        </IconButton>
      </Box>
      {!isCover && (
        <Box position="absolute" bottom="1" left="1" right="1">
          <Button
            type="button"
            size="xs"
            variant="subtle"
            bg="bg.default"
            w="full"
            onClick={() => onMakeCover(a.id)}
          >
            {t.marketplace.form.makeCover}
          </Button>
        </Box>
      )}
    </Box>
  );
}
