import {
  ListingCategory,
  ListingCondition,
  ListingPriceType,
  ListingStatus,
} from "@/api/openapi-schema";
import { Translations } from "@/lib/i18n";

export function categoryOptions(t: Translations) {
  return Object.values(ListingCategory).map((value) => ({
    value,
    label: t.marketplace.categories[value],
  }));
}

export function conditionOptions(t: Translations) {
  return Object.values(ListingCondition).map((value) => ({
    value,
    label: t.marketplace.conditions[value],
  }));
}

export function priceTypeOptions(t: Translations) {
  return Object.values(ListingPriceType).map((value) => ({
    value,
    label: t.marketplace.priceTypes[value],
  }));
}

export function statusOptions(t: Translations) {
  return Object.values(ListingStatus).map((value) => ({
    value,
    label: t.marketplace.statuses[value],
  }));
}
