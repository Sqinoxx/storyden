import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { AdminSettings } from "@/lib/settings/settings";

export type Props = {
  settings: AdminSettings;
};

export const DEFAULT_MAX_UPLOAD_SIZE_MB = 50;

export const FormSchema = z.object({
  maxUploadSizeMb: z.number().min(1).max(1024),
});
export type Form = z.infer<typeof FormSchema>;

export function useAssetSettings({ settings }: Props) {
  const { revalidate, updateSettings } = useSettingsMutation();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      maxUploadSizeMb:
        settings.services?.assets?.max_upload_size_mb ??
        DEFAULT_MAX_UPLOAD_SIZE_MB,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            assets: {
              max_upload_size_mb: data.maxUploadSizeMb,
            },
          },
        });
      },
      {
        promiseToast: {
          loading: "Saving settings...",
          success: "Settings saved",
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  });

  return {
    register: form.register,
    control: form.control,
    formState: form.formState,
    onSubmit,
  };
}
