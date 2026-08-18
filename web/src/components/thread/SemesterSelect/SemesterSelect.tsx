"use client";

import { useMemo } from "react";
import { createListCollection } from "@ark-ui/react";
import { FieldValues } from "react-hook-form";

import {
  SelectField,
  SelectFieldProps,
} from "@/components/ui/form/SelectField";
import { useTranslation } from "@/lib/i18n";
import {
  formatTermLabel,
  selectableTerms,
  termKey,
} from "@/lib/thread/semester";

export function SemesterSelect<T extends FieldValues>(
  props: Omit<SelectFieldProps<T, any>, "collection" | "placeholder">,
) {
  const t = useTranslation();

  const collection = useMemo(
    () =>
      createListCollection({
        items: selectableTerms(new Date()).map((term) => ({
          value: termKey(term),
          label: formatTermLabel(term),
        })),
      }),
    [],
  );

  return (
    <SelectField
      control={props.control}
      name={props.name}
      placeholder={t.thread.semesterPlaceholder}
      collection={collection}
    />
  );
}
