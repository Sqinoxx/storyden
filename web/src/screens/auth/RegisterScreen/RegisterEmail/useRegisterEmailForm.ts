"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import * as z from "zod";

import { handle } from "@/api/client";
import { authEmailPasswordSignup } from "@/api/openapi-client/auth";
import { PasswordSchema, UsernameSchema } from "@/lib/auth/schemas";
import { deriveError } from "@/utils/error";

const FormSchema = z.object({
  handle: UsernameSchema,
  email: z.string().email(),
  password: PasswordSchema,
});
type Form = z.infer<typeof FormSchema>;

type Props = {
  invitationID?: string;
};

export function useRegisterEmailForm({ invitationID }: Props) {
  const searchParams = useSearchParams();
  const returnURL = searchParams.get("return_url") ?? "/d";

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
  });

  const handleSubmit = form.handleSubmit(async (payload: Form) => {
    await handle(
      async () => {
        await authEmailPasswordSignup(payload, {
          invitation_id: invitationID,
        });

        // Hard redirect: forces a full page reload so the server renders the
        // authenticated state with the new session cookie immediately.
        // Avoids the Next.js client-cache race condition that required a second click.
        window.location.href = returnURL;
      },
      {
        errorToast: false,
        onError: async (error) => {
          form.setError("root", { message: deriveError(error) });
        },
      },
    );
  });

  return {
    form,
    handlers: {
      handleSubmit,
    },
  };
}

