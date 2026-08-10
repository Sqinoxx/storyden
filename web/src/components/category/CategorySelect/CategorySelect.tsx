"use client";

import { FieldValues, useFormState } from "react-hook-form";

import { ErrorTooltip } from "@/components/ui/ErrorTooltip";
import {
  SelectField,
  SelectFieldProps,
} from "@/components/ui/form/SelectField";
import { HStack } from "@/styled-system/jsx";

import { useCategorySelect } from "./useCategorySelect";
import { useTranslation } from "@/lib/i18n";

export function CategorySelect<T extends FieldValues>(
  props: Omit<SelectFieldProps<T, any>, "collection" | "placeholder">,
) {
  const result = useCategorySelect();
  const { ready, collection, error } = result;
  const t = useTranslation();

  const { errors } = useFormState({ control: props.control, name: props.name });
  const fieldError = errors[props.name]?.message as string | undefined;

  return (
    <HStack gap="2" alignItems="center">
      <SelectField
        control={props.control}
        name={props.name}
        disabled={!ready}
        placeholder={ready ? t.category.category : t.category.loadingCategories}
        collection={collection}
      />
      <ErrorTooltip error={fieldError ?? error} />
    </HStack>
  );
}
