import { toast } from "sonner";

import { ProblemType, isProblemType } from "./problems";

export const UNVERIFIED_EMAIL_MESSAGE =
  "Bitte bestätige deine E-Mail-Adresse, um alle Funktionen zu nutzen.";

/**
 * Requests a fresh verification code. Without an address the signed-in
 * account's unverified address is used.
 *
 * The generated client is imported lazily because it imports `handle` from
 * `client.ts`, which needs this module — a static import either way would be a
 * cycle.
 */
export async function resendVerificationEmail(email?: string) {
  const { authEmailVerifyResend } = await import("./openapi-client/auth");

  await authEmailVerifyResend(email ? { email } : {});
}

/**
 * Replaces the raw permission error with a friendly prompt and a one-click way
 * out of it. Returns true when the error was handled here.
 */
export function handleUnverifiedEmailError(error: unknown): boolean {
  if (!isProblemType(error, ProblemType.EmailNotVerified)) {
    return false;
  }

  toast.error(UNVERIFIED_EMAIL_MESSAGE, {
    action: {
      label: "Neue Bestätigungs-E-Mail senden",
      onClick: () => {
        void toast.promise(resendVerificationEmail(), {
          loading: "Bestätigungs-E-Mail wird gesendet...",
          success: "Bestätigungs-E-Mail gesendet. Schau in dein Postfach.",
          error: "Die Bestätigungs-E-Mail konnte nicht gesendet werden.",
        });
      },
    },
  });

  return true;
}
