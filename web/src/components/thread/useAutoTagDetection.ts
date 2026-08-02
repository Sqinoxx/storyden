import { useEffect, useRef } from "react";
import { Control, FieldValues, Path, useWatch } from "react-hook-form";

import { useTagList } from "@/api/openapi-client/tags";
import { Asset } from "@/api/openapi-schema";

export type AttachmentItem =
  | Asset
  | { filename?: string; name?: string }
  | string;

interface UseAutoTagDetectionProps<T extends FieldValues> {
  control?: Control<T>;
  currentTags: string[];
  onChange: (tags: string[]) => void;
  enabled?: boolean;
  attachments?: AttachmentItem[];
}

export function useAutoTagDetection<T extends FieldValues>({
  control,
  currentTags,
  onChange,
  enabled = true,
  attachments = [],
}: UseAutoTagDetectionProps<T>) {
  const { data: tagListData } = useTagList();

  const title = useWatch({ control, name: "title" as Path<T> }) as
    | string
    | undefined;
  const body = useWatch({ control, name: "body" as Path<T> }) as
    | string
    | undefined;
  const watchedAttachments = useWatch({
    control,
    name: "attachments" as Path<T>,
  }) as AttachmentItem[] | undefined;
  const watchedFiles = useWatch({
    control,
    name: "files" as Path<T>,
  }) as AttachmentItem[] | undefined;

  const manuallyRemovedTagsRef = useRef<Set<string>>(new Set());
  const prevTagsRef = useRef<string[]>(currentTags || []);

  // Track manually removed or manually added tags
  useEffect(() => {
    if (!enabled) return;
    const tags = currentTags || [];
    const prevTags = prevTagsRef.current;

    for (const prevTag of prevTags) {
      if (!tags.includes(prevTag)) {
        manuallyRemovedTagsRef.current.add(prevTag.toLowerCase());
      }
    }
    for (const tag of tags) {
      if (!prevTags.includes(tag)) {
        manuallyRemovedTagsRef.current.delete(tag.toLowerCase());
      }
    }
    prevTagsRef.current = tags;
  }, [currentTags, enabled]);

  useEffect(() => {
    if (!enabled || !tagListData?.tags || tagListData.tags.length === 0) {
      return;
    }

    // Combine all attachment filename sources
    const allAttachments = [
      ...attachments,
      ...(watchedAttachments || []),
      ...(watchedFiles || []),
    ];

    const attachmentFilenames = allAttachments
      .map((item) => {
        if (!item) return "";
        if (typeof item === "string") return item;
        if ("filename" in item && item.filename) return item.filename;
        if ("name" in item && item.name) return item.name;
        return "";
      })
      .filter(Boolean)
      .join(" ");

    // Strip HTML markup for inner text
    const cleanBody = (body || "").replace(/<[^>]*>/g, " ");

    // Include title, raw body (with html attributes like alt/data-filename), cleanBody, and attachment filenames
    const combinedText = `${title || ""} ${body || ""} ${cleanBody} ${attachmentFilenames}`.toLowerCase();
    const normalizedText = combinedText.replace(/[-_./\\,]/g, " ");

    const existingTags = currentTags || [];
    const existingTagsLower = existingTags.map((t) => t.toLowerCase());

    const tagsToAdd: string[] = [];

    for (const tagObj of tagListData.tags) {
      const tagName = tagObj.name;
      if (!tagName || tagName.trim().length < 2) continue;

      const tagNameLower = tagName.toLowerCase();

      // Case-insensitive check if tag is 1:1 or contained within a word/filename in text
      const isMatched =
        combinedText.includes(tagNameLower) ||
        normalizedText.includes(tagNameLower);

      if (isMatched) {
        if (
          !existingTagsLower.includes(tagNameLower) &&
          !manuallyRemovedTagsRef.current.has(tagNameLower)
        ) {
          tagsToAdd.push(tagName);
        }
      } else {
        // Text/filenames no longer contain tag word, so reset manual removal flag
        manuallyRemovedTagsRef.current.delete(tagNameLower);
      }
    }

    if (tagsToAdd.length > 0) {
      onChange([...existingTags, ...tagsToAdd]);
    }
  }, [
    title,
    body,
    attachments,
    watchedAttachments,
    watchedFiles,
    tagListData,
    currentTags,
    onChange,
    enabled,
  ]);
}
