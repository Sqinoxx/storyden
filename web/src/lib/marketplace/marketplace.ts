import { Listing, ListingPriceType } from "@/api/openapi-schema";
import { Translations } from "@/lib/i18n";

const euro = new Intl.NumberFormat("de-AT", {
  style: "currency",
  currency: "EUR",
  minimumFractionDigits: 0,
  maximumFractionDigits: 2,
});

export function formatCents(cents: number) {
  return euro.format(cents / 100);
}

export function formatListingPrice(
  t: Translations,
  listing: Pick<Listing, "price" | "price_type">,
) {
  if (listing.price_type === ListingPriceType.free) {
    return t.marketplace.free;
  }

  const price = formatCents(listing.price);
  if (listing.price_type === ListingPriceType.negotiable) {
    return `${price} ${t.marketplace.negotiableShort}`;
  }

  return price;
}

export function eurosToCents(euros: number) {
  return Math.round(euros * 100);
}

export function centsToEuros(cents: number) {
  return cents / 100;
}

export function formatDate(iso: string) {
  return new Date(iso).toLocaleDateString("de-AT", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

export function whatsappURL(phone: string, message: string) {
  const digits = phone.replace(/[^0-9]/g, "");
  return `https://wa.me/${digits}?text=${encodeURIComponent(message)}`;
}

export function telURL(phone: string) {
  return `tel:${phone}`;
}

export function listingURL(listing: Pick<Listing, "slug">) {
  return `/flohmarkt/${listing.slug}`;
}
