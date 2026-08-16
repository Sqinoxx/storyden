"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import * as z from "zod";

import { handle } from "@/api/client";
import { accountUpdate } from "@/api/openapi-client/accounts";
import { authEmailPasswordSignup } from "@/api/openapi-client/auth";
import { PasswordSchema, UsernameSchema } from "@/lib/auth/schemas";
import {
  ACADEMIC_META_KEY,
  MAX_SEMESTER,
  MIN_SEMESTER,
} from "@/lib/profile/academic";
import { deriveError } from "@/utils/error";

const FormSchema = z.object({
  handle: UsernameSchema,
  email: z.string().email(),
  password: PasswordSchema,
  semester: z.coerce.number().int().min(MIN_SEMESTER).max(MAX_SEMESTER),
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

        try {
          await accountUpdate({
            meta: { [ACADEMIC_META_KEY]: { semester: payload.semester } },
          });
        } catch {
          // Best-effort: the account exists either way, semester can be set later from the profile page.
        }

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

