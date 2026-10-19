"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAccountGet } from "@/api/openapi-client/accounts";
import { isUnauthenticatedError } from "@/auth/unauthenticated";

import { Unready } from "@/components/site/Unready";
import { GenericError } from "@/screens/errors/GenericError";
import { deriveError } from "@/utils/error";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const { replace } = useRouter();
  const account = useAccountGet();

  const settled = !account.isLoading;
  const loggedOut =
    settled && (!account.data || isUnauthenticatedError(account.error));

  useEffect(() => {
    console.error("(dashboard) error caught:", error);
  }, [error]);

  useEffect(() => {
    if (loggedOut) {
      replace("/login");
    }
  }, [loggedOut, replace]);

  if (!settled || loggedOut) {
    return <Unready />;
  }

  return <GenericError reset={reset} message={deriveError(error)} />;
}
