import { zodResolver } from "@hookform/resolvers/zod";
import { useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useTranslation } from "@/lib/i18n";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { AdminSettings } from "@/lib/settings/settings";

export type Props = {
  settings: AdminSettings;
};

export const DEFAULT_LISTING_DURATION_DAYS = 90;

export const FormSchema = z.object({
  enabled: z.boolean(),
  listingDurationDays: z.number().int().min(1).max(365),
  categories: z
    .array(
      z.object({
        id: z.string(),
        name: z.string().trim().min(1).max(64),
      }),
    )
    .min(1),
});
export type Form = z.infer<typeof FormSchema>;

export function useMarketSettings({ settings }: Props) {
  const t = useTranslation();
  const { revalidate, updateSettings } = useSettingsMutation();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      enabled: settings.services?.market?.enabled ?? false,
      listingDurationDays:
        settings.services?.market?.listing_duration_days ??
        DEFAULT_LISTING_DURATION_DAYS,
      categories: settings.services?.market?.categories ?? [],
    },
  });

  const categories = useFieldArray({
    control: form.control,
    name: "categories",
    keyName: "key",
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            market: {
              enabled: data.enabled,
              listing_duration_days: data.listingDurationDays,
              categories: withCategoryIDs(data.categories),
            },
          },
        });
      },
      {
        promiseToast: {
          loading: t.toasts.savingSettings,
          success: t.toasts.settingsSaved,
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  });

  return {
    control: form.control,
    register: form.register,
    formState: form.formState,
    categories,
    onSubmit,
  };
}

function slugify(name: string) {
  const slug = name
    .toLowerCase()
    .replace(/ä/g, "ae")
    .replace(/ö/g, "oe")
    .replace(/ü/g, "ue")
    .replace(/ß/g, "ss")
    .normalize("NFKD")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, 60)
    .replace(/-+$/, "");

  return slug || "kategorie";
}

function withCategoryIDs(categories: Form["categories"]) {
  const taken = new Set(categories.map((c) => c.id).filter(Boolean));

  return categories.map((c) => {
    if (c.id) return { id: c.id, name: c.name.trim() };

    const base = slugify(c.name);
    let id = base;
    for (let n = 2; taken.has(id); n++) id = `${base}-${n}`;
    taken.add(id);

    return { id, name: c.name.trim() };
  });
}
