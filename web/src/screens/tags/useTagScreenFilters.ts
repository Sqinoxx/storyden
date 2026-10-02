import { parseAsString, parseAsStringLiteral, useQueryStates } from "nuqs";

import {
  TAG_ITEM_KINDS,
  TAG_ITEM_SORTS,
  TagItemFilters,
} from "./filterTagItems";

const parsers = {
  q: parseAsString.withDefault(""),
  kind: parseAsStringLiteral(TAG_ITEM_KINDS),
  category: parseAsString,
  author: parseAsString,
  sort: parseAsStringLiteral(TAG_ITEM_SORTS).withDefault("newest"),
};

export function useTagScreenFilters() {
  const [filters, setFilters] = useQueryStates(parsers, {
    clearOnDefault: true,
  });

  function set(patch: Partial<TagItemFilters>) {
    setFilters(patch);
  }

  function reset() {
    setFilters({ q: null, kind: null, category: null, author: null });
  }

  return { filters: filters as TagItemFilters, set, reset };
}
