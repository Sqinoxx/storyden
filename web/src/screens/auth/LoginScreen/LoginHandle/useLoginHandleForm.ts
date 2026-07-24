"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import * as z from "zod";

import { authPasswordSignin } from "@/api/openapi-client/auth";
import { APIError } from "@/api/openapi-schema";
import { passkeyLogin } from "@/components/auth/webauthn/utils";
import { deriveError } from "@/utils/error";

import { ExistingPasswordSchema, UsernameSchema } from "@/lib/auth/schemas";
import { isWebauthnAvailable } from "@/lib/auth/webauthn";

export type Props = {
  webauthn: boolean;
};

const KindSchema = z.enum(["password", "webauthn"]);
type Kind = z.infer<typeof KindSchema>;

const FormSchema = z.object({
  identifier: UsernameSchema,
  token: z.string().optional(), // Validated properly during password submission
});
const FormPasswordSchema = z.object({
  identifier: UsernameSchema,
  token: ExistingPasswordSchema,
});
type Form = z.infer<typeof FormSchema>;

export function useLoginHandleForm() {
  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
    setError,
  } = useForm<Form>({
    resolver: zodResolver(FormSchema),
  });
  const searchParams = useSearchParams();
  const returnURL = searchParams.get("return_url") ?? "/d";

  const isWebauthnEnabled = isWebauthnAvailable();

  function handler(kind: Kind) {
    return handleSubmit(async (payload) => {
      switch (kind) {
        case "password":
          return await handlePassword(payload);
        case "webauthn":
          return await handleWebauthn(payload);
      }
    });
  }

  async function handlePassword(payload: Form) {
    const parsed = FormPasswordSchema.safeParse(payload);
    if (!parsed.success) {
      if (parsed.error.formErrors.fieldErrors.identifier) {
        setError("identifier", {
          message: parsed.error.formErrors.fieldErrors.identifier?.join(", "),
        });
      }

      if (parsed.error.formErrors.fieldErrors.token) {
        setError("token", {
          message: parsed.error.formErrors.fieldErrors.token?.join(", "),
        });
      }

      return;
    }

    try {
      await authPasswordSignin(parsed.data);
      // Hard redirect: forces a full page reload so the server renders the
      // authenticated state with the new session cookie immediately.
      // Avoids the Next.js client-cache race condition that required a second click.
      window.location.href = returnURL;
    } catch (e) {
      setError("root", { message: deriveError(e as APIError) });
    }
  }

  async function handleWebauthn(payload: Form) {
    try {
      await passkeyLogin(payload.identifier);
      window.location.href = returnURL;
    } catch (error) {
      setError("root", { message: deriveError(error) });
    }
  }

  return {
    form: {
      register,
      isWebauthnEnabled,
      handlePassword: handler("password"),
      handleWebauthn: handler("webauthn"),
      errors,
      isSubmitting,
    },
  };
}

