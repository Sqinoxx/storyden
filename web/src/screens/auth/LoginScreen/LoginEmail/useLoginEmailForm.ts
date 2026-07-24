"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import * as z from "zod";

import { handle } from "@/api/client";
import {
  authEmailPasswordSignin,
  authPasswordSignin,
} from "@/api/openapi-client/auth";
import {
  ExistingPasswordSchema,
  UsernameOrEmailSchema,
} from "@/lib/auth/schemas";

const FormSchema = z.object({
  identifier: UsernameOrEmailSchema,
  password: ExistingPasswordSchema,
});
type Form = z.infer<typeof FormSchema>;

export function useLoginEmailForm() {
  const searchParams = useSearchParams();
  const returnURL = searchParams.get("return_url") ?? "/d";

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
  });

  const handleSubmit = form.handleSubmit(async (payload: Form) => {
    await handle(
      async () => {
        const isEmail = payload.identifier.includes("@");

        if (isEmail) {
          await authEmailPasswordSignin({
            email: payload.identifier,
            password: payload.password,
          });
        } else {
          await authPasswordSignin({
            identifier: payload.identifier,
            token: payload.password,
          });
        }

        // Hard redirect: forces a full page reload so the server renders the
        // authenticated state with the new session cookie immediately.
        // Avoids the Next.js client-cache race condition that required a second click.
        window.location.href = returnURL;
      },
      {
        errorToast: false,
        onError: async (error) => {
          form.setError("root", {
            message:
              error instanceof Error ? error.message : "Login fehlgeschlagen.",
          });
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
