import { MarketCategory } from "@/api/openapi-schema";
import { useSettings } from "@/lib/settings/settings-client";

export function useMarketCategories() {
  const result = useSettings();
  const categories: MarketCategory[] = result.ready
    ? (result.settings.marketplace_categories ?? [])
    : [];

  const options = categories.map((c) => ({ value: c.id, label: c.name }));

  function label(id: string) {
    return categories.find((c) => c.id === id)?.name ?? id;
  }

  return { categories, options, label };
}
