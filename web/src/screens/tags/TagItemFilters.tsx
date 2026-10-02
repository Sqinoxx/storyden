import { SelectValueChangeDetails, createListCollection } from "@ark-ui/react";
import { useMemo } from "react";

import { CheckIcon } from "@/components/ui/icons/Check";
import { SearchIcon } from "@/components/ui/icons/Search";
import { SelectIcon } from "@/components/ui/icons/Select";
import { Input } from "@/components/ui/input";
import { InputGroup } from "@/components/ui/input-group";
import * as Select from "@/components/ui/select";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack } from "@/styled-system/jsx";

import {
  TAG_ITEM_KINDS,
  TAG_ITEM_SORTS,
  TagItemFacet,
  TagItemFilters as Filters,
  TagItemKind,
  TagItemSort,
} from "./filterTagItems";

const ALL = "__all";

type Props = {
  filters: Filters;
  onChange: (patch: Partial<Filters>) => void;
  categories: TagItemFacet[];
  authors: TagItemFacet[];
};

export function TagItemFilters({
  filters,
  onChange,
  categories,
  authors,
}: Props) {
  const t = useTranslation();

  const kindItems = useMemo(
    () => [
      { value: ALL, label: t.tags.filterAllTypes },
      ...TAG_ITEM_KINDS.map((value) => ({
        value,
        label: value === "thread" ? t.tags.threadsHeading : t.tags.pagesHeading,
      })),
    ],
    [t],
  );

  const sortLabels: Record<TagItemSort, string> = {
    newest: t.tags.sortNewest,
    oldest: t.tags.sortOldest,
    active: t.tags.sortActive,
    replies: t.tags.sortReplies,
    likes: t.tags.sortLikes,
    alphabetical: t.tags.sortAlphabetical,
  };
  const sortItems = TAG_ITEM_SORTS.map((value) => ({
    value,
    label: sortLabels[value],
  }));

  const pick = (value: string[]) => {
    const [selected] = value;
    return !selected || selected === ALL ? null : selected;
  };

  return (
    <LStack gap="3" w="full">
      <InputGroup
        w="full"
        startElement={<SearchIcon w="4" h="4" color="fg.muted" />}
      >
        <Input
          type="search"
          aria-label={t.tags.itemSearchPlaceholder}
          placeholder={t.tags.itemSearchPlaceholder}
          value={filters.q}
          onChange={(e) => onChange({ q: e.target.value })}
        />
      </InputGroup>

      <HStack gap="2" w="full" flexWrap="wrap">
        <FilterSelect
          label={t.tags.filterByType}
          items={kindItems}
          value={filters.kind ?? ALL}
          onChange={({ value }) =>
            onChange({ kind: pick(value) as TagItemKind | null })
          }
        />

        {categories.length > 0 && (
          <FilterSelect
            label={t.tags.filterByCategory}
            items={[
              { value: ALL, label: t.tags.filterAllCategories },
              ...categories,
            ]}
            value={filters.category ?? ALL}
            onChange={({ value }) => onChange({ category: pick(value) })}
          />
        )}

        {authors.length > 1 && (
          <FilterSelect
            label={t.tags.filterByAuthor}
            items={[{ value: ALL, label: t.tags.filterAllAuthors }, ...authors]}
            value={filters.author ?? ALL}
            onChange={({ value }) => onChange({ author: pick(value) })}
          />
        )}

        <FilterSelect
          label={t.tags.sortBy}
          items={sortItems}
          value={filters.sort}
          onChange={({ value }) =>
            onChange({ sort: (value[0] as TagItemSort) ?? "newest" })
          }
        />
      </HStack>
    </LStack>
  );
}

function FilterSelect({
  label,
  items,
  value,
  onChange,
}: {
  label: string;
  items: TagItemFacet[];
  value: string;
  onChange: (details: SelectValueChangeDetails) => void;
}) {
  const collection = useMemo(() => createListCollection({ items }), [items]);

  return (
    <Select.Root
      size="sm"
      width="auto"
      collection={collection}
      value={[value]}
      positioning={{ sameWidth: false }}
      onValueChange={onChange}
    >
      <Select.Control>
        <Select.Trigger aria-label={label}>
          <Select.ValueText placeholder={label} />
          <SelectIcon />
        </Select.Trigger>
      </Select.Control>
      <Select.Positioner>
        <Select.Content>
          {collection.items.map((item) => (
            <Select.Item key={item.value} item={item}>
              <Select.ItemText>{item.label}</Select.ItemText>
              <Select.ItemIndicator>
                <CheckIcon />
              </Select.ItemIndicator>
            </Select.Item>
          ))}
        </Select.Content>
      </Select.Positioner>
    </Select.Root>
  );
}
