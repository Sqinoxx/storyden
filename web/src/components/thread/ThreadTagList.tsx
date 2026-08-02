"use client";

import { useState } from "react";
import { Control, Controller, ControllerProps, FieldValues } from "react-hook-form";

import { handle } from "@/api/client";
import { tagList } from "@/api/openapi-client/tags";
import { TagNameList, TagReferenceList } from "@/api/openapi-schema";
import { TagBadgeList } from "@/components/tag/TagBadgeList";
import {
  MultiSelectPicker,
  MultiSelectPickerItem,
} from "@/components/ui/MultiSelectPicker";
import { useTranslation } from "@/lib/i18n";

import { ButtonProps } from "@/components/ui/button";
import { AttachmentItem, useAutoTagDetection } from "./useAutoTagDetection";

export type Props = {
  editing: boolean;
  initialTags?: TagReferenceList;
  onChange: (tags: TagNameList) => Promise<void>;
  triggerProps?: ButtonProps;
};

export function ThreadTagList(props: Props) {
  const [queryResults, setQueryResults] = useState<MultiSelectPickerItem[]>(
    [],
  );
  const t = useTranslation();

  const currentTags: MultiSelectPickerItem[] =
    props.initialTags?.map((t) => ({
      label: t.name,
      value: t.name,
    })) ?? [];

  function handleQuery(q: string) {
    handle(async () => {
      const { tags } = await tagList({ q });
      const filtered = tags.filter(
        (t) => !currentTags.some((ct) => ct.value === t.name),
      );
      setQueryResults(
        filtered.map((t) => ({
          label: t.name,
          value: t.name,
        })),
      );
    });
  }

  async function handleChange(items: MultiSelectPickerItem[]) {
    const tagNames = items.map((item) => item.value);
    await props.onChange(tagNames);
  }

  if (props.editing) {
    return (
      <MultiSelectPicker
        value={currentTags}
        onChange={handleChange}
        onQuery={handleQuery}
        queryResults={queryResults}
        allowNewValues={true}
        inputPlaceholder={t.editor.addTags}
        autoColour={true}
        size="sm"
        triggerProps={
          props.triggerProps ?? {
            w: "auto",
            minW: "[140px]",
            maxW: "full",
          }
        }
      />
    );
  }

  if (props.initialTags?.length === 0) {
    return null;
  }

  return <TagBadgeList tags={props.initialTags ?? []} />;
}

type TagListFieldProps<T extends FieldValues> = Omit<
  ControllerProps<T>,
  "render"
> & {
  initialTags?: TagReferenceList;
  triggerProps?: ButtonProps;
  autoDetect?: boolean;
  attachments?: AttachmentItem[];
};

export function TagListField<T extends FieldValues>({
  control,
  name,
  initialTags,
  triggerProps,
  autoDetect = true,
  attachments,
}: TagListFieldProps<T>) {
  return (
    <Controller<T>
      render={({ field }) => {
        async function handleChange(tags: string[]) {
          field.onChange(tags);
        }

        const fieldTags =
          field.value?.map((name: string) => ({ name })) || initialTags || [];

        return (
          <TagListFieldInternal
            control={control}
            fieldValue={field.value || []}
            fieldTags={fieldTags}
            onChange={handleChange}
            triggerProps={triggerProps}
            autoDetect={autoDetect}
            attachments={attachments}
          />
        );
      }}
      control={control}
      name={name}
    />
  );
}

function TagListFieldInternal<T extends FieldValues>({
  control,
  fieldValue,
  fieldTags,
  onChange,
  triggerProps,
  autoDetect = true,
  attachments,
}: {
  control?: Control<T>;
  fieldValue: string[];
  fieldTags: TagReferenceList;
  onChange: (tags: string[]) => Promise<void>;
  triggerProps?: ButtonProps;
  autoDetect?: boolean;
  attachments?: AttachmentItem[];
}) {
  useAutoTagDetection({
    control,
    currentTags: fieldValue,
    onChange: (nextTags) => {
      onChange(nextTags);
    },
    enabled: autoDetect,
    attachments,
  });

  return (
    <ThreadTagList
      editing={true}
      initialTags={fieldTags}
      onChange={onChange}
      triggerProps={triggerProps}
    />
  );
}
