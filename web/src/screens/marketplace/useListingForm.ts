import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { listingCreate, listingUpdate } from "@/api/openapi-client/listings";
import {
  Asset,
  Listing,
  ListingCondition,
  ListingPriceType,
} from "@/api/openapi-schema";
import { useMarketCategories } from "@/components/marketplace/useMarketCategories";
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
  category: z.string().min(1),
  condition: z.nativeEnum(ListingCondition),
});
export type Form = z.infer<typeof FormSchema>;

export type Props = {
  listing?: Listing;
};

export function useListingForm({ listing }: Props) {
  const t = useTranslation();
  const { categories } = useMarketCategories();
  const router = useRouter();
  const [assets, setAssets] = useState<Asset[]>(listing?.assets ?? []);
  const [uploading, setUploading] = useState(false);
  const [publishing, setPublishing] = useState(false);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      title: listing?.title ?? "",
      description: listing?.description ?? "",
      price: listing ? centsToEuros(listing.price) : 0,
      price_type: listing?.price_type ?? ListingPriceType.negotiable,
      category: listing?.category ?? categories[0]?.id ?? "",
      condition: listing?.condition ?? ListingCondition.used,
    },
  });

  const priceType = form.watch("price_type");

  const firstCategory = categories[0]?.id;
  useEffect(() => {
    if (firstCategory && !form.getValues("category")) {
      form.setValue("category", firstCategory);
    }
  }, [firstCategory, form]);

  const onSubmit = form.handleSubmit(async (data) => {
    setPublishing(true);
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
    } else {
      setPublishing(false);
    }
  });

  return {
    form,
    priceType,
    assets,
    setAssets,
    uploading,
    setUploading,
    publishing,
    onSubmit,
  };
}
