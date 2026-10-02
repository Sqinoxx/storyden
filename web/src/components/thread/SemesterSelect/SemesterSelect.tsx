"use client";

import { useMemo } from "react";
import { createListCollection } from "@ark-ui/react";
import { FieldValues } from "react-hook-form";

import {
  SelectField,
  SelectFieldProps,
} from "@/components/ui/form/SelectField";
import { useTranslation } from "@/lib/i18n";
import { useSettings } from "@/lib/settings/settings-client";
import {
  formatSelectableTermLabel,
  selectableTerms,
  termKey,
} from "@/lib/thread/semester";

export function SemesterSelect<T extends FieldValues>(
  props: Omit<SelectFieldProps<T, any>, "collection" | "placeholder">,
) {
  const t = useTranslation();

  const settingsResult = useSettings();
  const legacyLabel = settingsResult.ready
    ? settingsResult.settings.metadata.semester.legacyLabel
    : "";

  const collection = useMemo(() => {
    const now = new Date();

    return createListCollection({
      items: selectableTerms(now).map((term) => ({
        value: termKey(term),
        label: formatSelectableTermLabel(term, now, legacyLabel),
      })),
    });
  }, [legacyLabel]);

  return (
    <SelectField
      control={props.control}
      name={props.name}
      placeholder={t.thread.semesterPlaceholder}
      collection={collection}
    />
  );
}
