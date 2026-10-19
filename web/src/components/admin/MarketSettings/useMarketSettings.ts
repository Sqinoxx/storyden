import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
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
  listingDurationDays: z.number().int().min(1).max(365),
});
export type Form = z.infer<typeof FormSchema>;

export function useMarketSettings({ settings }: Props) {
  const t = useTranslation();
  const { revalidate, updateSettings } = useSettingsMutation();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      listingDurationDays:
        settings.services?.market?.listing_duration_days ??
        DEFAULT_LISTING_DURATION_DAYS,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            market: {
              listing_duration_days: data.listingDurationDays,
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
    formState: form.formState,
    onSubmit,
  };
}
