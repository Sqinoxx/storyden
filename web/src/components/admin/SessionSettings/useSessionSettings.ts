import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { AdminSettings } from "@/lib/settings/settings";

export type Props = {
  settings: AdminSettings;
};

export const DEFAULT_IDLE_TIMEOUT_MINUTES = 60;
export const DEFAULT_REMEMBER_ME_DAYS = 30;
export const DEFAULT_REMEMBER_ME_MAX_DAYS = 365;

export const FormSchema = z.object({
  idleTimeoutMinutes: z.number().int().min(1).max(43_200),
  rememberMeDefaultDays: z.number().int().min(1).max(365),
  rememberMeMaxDays: z.number().int().min(1).max(365),
});
export type Form = z.infer<typeof FormSchema>;

export function useAdminSessionSettings({ settings }: Props) {
  const { revalidate, updateSettings } = useSettingsMutation();

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      idleTimeoutMinutes:
        settings.services?.session?.idle_timeout_minutes ??
        DEFAULT_IDLE_TIMEOUT_MINUTES,
      rememberMeDefaultDays:
        settings.services?.session?.remember_me_default_days ??
        DEFAULT_REMEMBER_ME_DAYS,
      rememberMeMaxDays:
        settings.services?.session?.remember_me_max_days ??
        DEFAULT_REMEMBER_ME_MAX_DAYS,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            session: {
              idle_timeout_minutes: data.idleTimeoutMinutes,
              remember_me_default_days: data.rememberMeDefaultDays,
              remember_me_max_days: data.rememberMeMaxDays,
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
    control: form.control,
    formState: form.formState,
    onSubmit,
  };
}
