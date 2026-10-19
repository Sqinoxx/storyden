import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { listingCreate, listingUpdate } from "@/api/openapi-client/listings";
import {
  Asset,
  Listing,
  ListingCategory,
  ListingCondition,
  ListingPriceType,
} from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import {
  centsToEuros,
  eurosToCents,
  listingURL,
} from "@/lib/marketplace/marketplace";

export const FormSchema = z.object({
  title: z.string().trim().min(3).max(100),
  description: z.string().max(5000),
  price: z.coerce.number().min(0).max(1000000),
  price_type: z.nativeEnum(ListingPriceType),
  category: z.nativeEnum(ListingCategory),
  condition: z.nativeEnum(ListingCondition),
  location: z.string().trim().max(100),
});
export type Form = z.infer<typeof FormSchema>;

export type Props = {
  listing?: Listing;
};

export function useListingForm({ listing }: Props) {
  const t = useTranslation();
  const router = useRouter();
  const [assets, setAssets] = useState<Asset[]>(listing?.assets ?? []);
  const [uploading, setUploading] = useState(false);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      title: listing?.title ?? "",
      description: listing?.description ?? "",
      price: listing ? centsToEuros(listing.price) : 0,
      price_type: listing?.price_type ?? ListingPriceType.negotiable,
      category: listing?.category ?? ListingCategory.other,
      condition: listing?.condition ?? ListingCondition.used,
      location: listing?.location ?? "",
    },
  });

  const priceType = form.watch("price_type");

  const onSubmit = form.handleSubmit(async (data) => {
    const body = {
      title: data.title,
      description: data.description,
      price:
        data.price_type === ListingPriceType.free
          ? 0
          : eurosToCents(data.price),
      price_type: data.price_type,
      category: data.category,
      condition: data.condition,
      location: data.location,
      asset_ids: assets.map((a) => a.id),
    };

    const saved = await handle(
      async () => {
        if (listing) {
          return await listingUpdate(listing.slug, body);
        }
        return await listingCreate(body);
      },
      {
        promiseToast: listing
          ? {
              loading: t.marketplace.form.saving,
              success: t.marketplace.form.saved,
            }
          : {
              loading: t.marketplace.form.creating,
              success: t.marketplace.form.created,
            },
      },
    );

    if (saved) {
      router.push(listingURL(saved));
    }
  });

  return {
    form,
    priceType,
    assets,
    setAssets,
    uploading,
    setUploading,
    onSubmit,
  };
}
