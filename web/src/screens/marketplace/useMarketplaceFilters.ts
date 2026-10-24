import {
  parseAsFloat,
  parseAsInteger,
  parseAsString,
  parseAsStringEnum,
  useQueryStates,
} from "nuqs";

import {
  ListingCondition,
  ListingListParams,
  ListingPriceType,
  ListingScopeQueryParameter,
  ListingSortQueryParameter,
} from "@/api/openapi-schema";
import { eurosToCents } from "@/lib/marketplace/marketplace";

export const filterParsers = {
  q: parseAsString.withDefault(""),
  category: parseAsString,
  condition: parseAsStringEnum(Object.values(ListingCondition)),
  price_type: parseAsStringEnum(Object.values(ListingPriceType)),
  price_min: parseAsFloat,
  price_max: parseAsFloat,
  sort: parseAsStringEnum(Object.values(ListingSortQueryParameter)),
  scope: parseAsStringEnum(Object.values(ListingScopeQueryParameter)),
  page: parseAsInteger.withDefault(1),
};

export function useMarketplaceFilters() {
  const [filters, setFilters] = useQueryStates(filterParsers, {
    clearOnDefault: true,
  });

  const params: ListingListParams = {
    q: filters.q || undefined,
    category: filters.category ?? undefined,
    condition: filters.condition ?? undefined,
    price_type: filters.price_type ?? undefined,
    price_min:
      filters.price_min != null
        ? String(eurosToCents(filters.price_min))
        : undefined,
    price_max:
      filters.price_max != null
        ? String(eurosToCents(filters.price_max))
        : undefined,
    sort: filters.sort ?? undefined,
    scope: filters.scope ?? undefined,
    page: filters.page > 1 ? String(filters.page) : undefined,
  };

  function update(next: Partial<typeof filters>) {
    setFilters({ ...next, page: next.page ?? null });
  }

  return { filters, params, update };
}
