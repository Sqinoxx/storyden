import { useEffect, useMemo, useRef } from "react";
import { Control, FieldValues, Path, useWatch } from "react-hook-form";

import { useCategoryList } from "@/api/openapi-client/categories";
import { useTagList } from "@/api/openapi-client/tags";
import { Asset, Category } from "@/api/openapi-schema";

export const DEFAULT_NEW_THREAD_TAGS = ["Altklausur"];

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
  defaultTags?: string[];
  categoryHint?: string;
  resetKey?: string;
}

export function useAutoTagDetection<T extends FieldValues>({
  control,
  currentTags,
  onChange,
  enabled = true,
  attachments = [],
  defaultTags,
  categoryHint,
  resetKey,
}: UseAutoTagDetectionProps<T>) {
  const { data: tagListData } = useTagList();
  const { data: categoryListData } = useCategoryList();

  const title = useWatch({ control, name: "title" as Path<T> }) as
    | string
    | undefined;
  const body = useWatch({ control, name: "body" as Path<T> }) as
    | string
    | undefined;
  const watchedCategory = useWatch({
    control,
    name: "category" as Path<T>,
  }) as string | undefined;
  const watchedAttachments = useWatch({
    control,
    name: "attachments" as Path<T>,
  }) as AttachmentItem[] | undefined;
  const watchedFiles = useWatch({
    control,
    name: "files" as Path<T>,
  }) as AttachmentItem[] | undefined;

  const manuallyRemovedTagsRef = useRef<Set<string>>(new Set());
  const categoryAddedTagsRef = useRef<Set<string>>(new Set());
  const prevTagsRef = useRef<string[]>(currentTags || []);

  const categoryKeys = useMemo(
    () =>
      categoryChainKeys(
        categoryListData?.categories ?? [],
        watchedCategory || categoryHint,
      ),
    [categoryListData, watchedCategory, categoryHint],
  );

  const defaultTagsLower = useMemo(
    () => new Set((defaultTags ?? []).map((t) => t.toLowerCase())),
    [defaultTags],
  );

  useEffect(() => {
    manuallyRemovedTagsRef.current.clear();
    categoryAddedTagsRef.current.clear();
    prevTagsRef.current = currentTags || [];
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [resetKey]);

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
    if (!enabled || !tagListData?.tags) {
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
    const tagsToRemove = new Set<string>();
    const knownTagsLower = new Set<string>();

    const shouldAdd = (tagNameLower: string) =>
      !existingTagsLower.includes(tagNameLower) &&
      !manuallyRemovedTagsRef.current.has(tagNameLower) &&
      !tagsToAdd.some((t) => t.toLowerCase() === tagNameLower);

    for (const tagObj of tagListData.tags) {
      const tagName = tagObj.name;
      if (!tagName || tagName.trim().length < 2) continue;

      const tagNameLower = tagName.toLowerCase();
      knownTagsLower.add(tagNameLower);

      // Case-insensitive check if tag is 1:1 or contained within a word/filename in text
      const isTextMatched =
        combinedText.includes(tagNameLower) ||
        normalizedText.includes(tagNameLower);
      const isCategoryMatched = categoryKeys.has(tagNameLower);
      const isDefault = defaultTagsLower.has(tagNameLower);

      if (isTextMatched || isCategoryMatched || isDefault) {
        if (shouldAdd(tagNameLower)) {
          tagsToAdd.push(tagName);
          if (isCategoryMatched && !isTextMatched && !isDefault) {
            categoryAddedTagsRef.current.add(tagNameLower);
          }
        }
      } else {
        // Text/filenames no longer contain tag word, so reset manual removal flag
        manuallyRemovedTagsRef.current.delete(tagNameLower);

        if (categoryAddedTagsRef.current.has(tagNameLower)) {
          categoryAddedTagsRef.current.delete(tagNameLower);
          tagsToRemove.add(tagNameLower);
        }
      }
    }

    for (const defaultTag of defaultTags ?? []) {
      const lower = defaultTag.toLowerCase();
      if (!knownTagsLower.has(lower) && shouldAdd(lower)) {
        tagsToAdd.push(defaultTag);
      }
    }

    if (tagsToAdd.length > 0 || tagsToRemove.size > 0) {
      onChange([
        ...existingTags.filter((t) => !tagsToRemove.has(t.toLowerCase())),
        ...tagsToAdd,
      ]);
    }
  }, [
    title,
    body,
    attachments,
    watchedAttachments,
    watchedFiles,
    tagListData,
    categoryKeys,
    defaultTags,
    defaultTagsLower,
    currentTags,
    onChange,
    enabled,
  ]);
}

function categoryChainKeys(categories: Category[], id: string | undefined) {
  const keys = new Set<string>();
  const byId = new Map(categories.map((c) => [c.id, c]));
  const visited = new Set<string>();

  let current = id ? byId.get(id) : undefined;
  while (current && !visited.has(current.id)) {
    visited.add(current.id);
    keys.add(current.name.toLowerCase());
    keys.add(current.slug.toLowerCase());
    current = current.parent ? byId.get(current.parent) : undefined;
  }

  return keys;
}
