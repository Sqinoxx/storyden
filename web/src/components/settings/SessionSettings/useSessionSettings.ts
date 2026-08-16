import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useProfileMutations } from "@/lib/profile/mutation";
import { Member } from "@/lib/settings/member-settings";

// Offered durations, in days. One hour is expressed as a fraction so the whole
// control stays a single unit.
export const SESSION_DURATION_OPTIONS = [
  { value: 1, labelKey: "oneDay" },
  { value: 7, labelKey: "sevenDays" },
  { value: 30, labelKey: "thirtyDays" },
  { value: 365, labelKey: "oneYear" },
] as const;

export const FormSchema = z.object({
  rememberMeDays: z.coerce.number().int().min(1).max(365),
});
export type Form = z.infer<typeof FormSchema>;

export type Props = {
  session: Member;
};

export function useSessionSettings({ session }: Props) {
  const { update, revalidate } = useProfileMutations(session.handle);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      rememberMeDays: session.meta.session.remember_me_days,
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        // Metadata is merged server-side, so only the session key is sent.
        await update({
          meta: {
            session: {
              remember_me_days: data.rememberMeDays,
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
    formState: form.formState,
    onSubmit,
  };
}
