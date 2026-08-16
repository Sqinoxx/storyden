import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { AdminSettings } from "@/lib/settings/settings";

export type Props = {
  settings: AdminSettings;
};

export const DEFAULT_REPLIES_PER_PAGE = 15;
export const DEFAULT_THREADS_PER_PAGE = 50;

export const FormSchema = z.object({
  repliesPerPage: z.number().min(1).max(200),
  threadsPerPage: z.number().min(1).max(100),
});
export type Form = z.infer<typeof FormSchema>;

export function useContentSettings({ settings }: Props) {
  const { revalidate, updateSettings } = useSettingsMutation();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      repliesPerPage:
        settings.services?.content?.replies_per_page ??
        DEFAULT_REPLIES_PER_PAGE,
      threadsPerPage:
        settings.services?.content?.threads_per_page ??
        DEFAULT_THREADS_PER_PAGE,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            content: {
              replies_per_page: data.repliesPerPage,
              threads_per_page: data.threadsPerPage,
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
