import { useMemo, useState } from "react";
import { Control, Controller, FieldValues, Path } from "react-hook-form";

import { useCategoryList } from "@/api/openapi-client/categories";
import { Category } from "@/api/openapi-schema";
import {
  MultiSelectPicker,
  MultiSelectPickerItem,
} from "@/components/ui/MultiSelectPicker";
import { categoryPathLabels } from "@/lib/category/tree";
import { useTranslation } from "@/lib/i18n";
import { deriveError } from "@/utils/error";

type Props<T extends FieldValues> = {
  control: Control<T>;
  name: Path<T>;
  excludeID: string;
};

const toItem =
  (labels: Map<string, string>) =>
  (c: Category): MultiSelectPickerItem => ({
    label: labels.get(c.id) ?? c.name,
    value: c.id,
    colour: c.colour,
  });

export function RelatedCategoriesField<T extends FieldValues>({
  control,
  name,
  excludeID,
}: Props<T>) {
  const t = useTranslation();
  const { data, error } = useCategoryList();
  const [query, setQuery] = useState("");

  const items = useMemo(() => {
    const categories = data?.categories ?? [];
    const labels = categoryPathLabels(categories);
    return categories
      .filter((c) => c.id !== excludeID)
      .map(toItem(labels))
      .sort((a, b) => a.label.localeCompare(b.label));
  }, [data, excludeID]);

  const results = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return items;
    return items.filter((i) => i.label.toLowerCase().includes(q));
  }, [items, query]);

  return (
    <Controller
      control={control}
      name={name}
      render={({ field }) => {
        const ids: string[] = field.value ?? [];
        const selected = ids
          .map((id) => items.find((i) => i.value === id))
          .filter((i): i is MultiSelectPickerItem => i !== undefined);

        return (
          <MultiSelectPicker
            value={selected}
            onChange={async (next) => field.onChange(next.map((i) => i.value))}
            onQuery={setQuery}
            queryResults={results}
            queryError={deriveError(error)}
            inputPlaceholder={t.category.relatedPlaceholder}
            resultsLabel={t.category.relatedResults}
            size="sm"
          />
        );
      }}
    />
  );
}
