"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useSearchParams } from "next/navigation";
import { useForm } from "react-hook-form";
import * as z from "zod";

import { authPasswordSignup } from "@/api/openapi-client/auth";
import { APIError } from "@/api/openapi-schema";
import { passkeyRegister } from "@/components/auth/webauthn/utils";
import { PasswordSchema, UsernameSchema } from "@/lib/auth/schemas";
import { isWebauthnAvailable } from "@/lib/auth/webauthn";
import { deriveError } from "@/utils/error";

export type Props = {
  webauthn: boolean;
  invitationID?: string;
};

const KindSchema = z.enum(["password", "webauthn"]);
type Kind = z.infer<typeof KindSchema>;

const FormSchema = z.object({
  identifier: UsernameSchema,
  token: z.string().optional(), // Validated properly during password submission
});
const FormPasswordSchema = z.object({
  identifier: UsernameSchema,
  token: PasswordSchema,
});
type Form = z.infer<typeof FormSchema>;

export function useRegisterHandleForm({ invitationID }: Props) {
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
      await authPasswordSignup(parsed.data, { invitation_id: invitationID });
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
      await passkeyRegister(payload.identifier);
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

