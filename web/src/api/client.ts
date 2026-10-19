import { toast } from "sonner";
import { mutate } from "swr";

import { isPublicPage, isUnauthenticatedError } from "@/auth/unauthenticated";
import { deriveError } from "@/utils/error";

import { Options, buildRequest, buildResult } from "./common";
import { ProblemType, isProblemType } from "./problems";
import { handleUnverifiedEmailError } from "./verification";

import { UNVERIFIED_EMAIL_MESSAGE } from "./verification";

export const fetcher = async <T>(opts: Options): Promise<T> => {
  const request = buildRequest({
    ...opts,
    // We use the browser default cache behaviour for the client side requests.
    // There's no revalidation set on the client as we're already using SWR for
    // that. The default cache behaviour will however make use of browser HTTP
    // Conditional Requests and ETag headers which some endpoints in Storyden
    // provide. This results in a mostly fast experience but it's slowed down a
    // bit by the server side behaviour (see server.ts comment for more info.)
    cache: "default",
  });

  const response = await fetch(request);

  // A 401 on any request may mean the session expired, so re-check the account
  // and let the auth provider send the user to the login page if it has.
  const isAccountCheck =
    opts.url === ACCOUNT_URL && (opts.method ?? "GET") === "GET";
  if (response.status === 401 && !isAccountCheck) {
    recheckSession();
  }

  return buildResult<T>(response);
};

const ACCOUNT_URL = "/accounts";

let sessionRecheck: Promise<unknown> | undefined;

function recheckSession() {
  if (sessionRecheck) return;

  sessionRecheck = mutate([ACCOUNT_URL]).finally(() => {
    sessionRecheck = undefined;
  });
}

const SESSION_EXPIRED_MESSAGE =
  "Deine Sitzung ist abgelaufen. Du wirst zur Anmeldung weitergeleitet.";

// While on a private page, a 401 is followed by a redirect to login so a
// short notice replaces the raw error.
function isSessionExpiry(error: unknown): boolean {
  return (
    isUnauthenticatedError(error) &&
    typeof window !== "undefined" &&
    !isPublicPage(window.location.pathname)
  );
}

type HandleArgs<T> = {
  onError?: (error: unknown) => Promise<void>;
  cleanup?: () => Promise<void>;
  promiseToast?: {
    loading: string;
    success: string;
  };
  action?: {
    label: string;
    onClick: (event: React.MouseEvent<HTMLButtonElement, MouseEvent>) => void;
  };
  errorToast?: boolean;
};

export async function handle<T>(
  fn: () => Promise<T>,
  args?: HandleArgs<T>,
): Promise<T | undefined> {
  const {
    onError,
    cleanup,
    promiseToast,
    errorToast = true,
    action,
  } = args ?? {};

  if (promiseToast) {
    return new Promise((resolve, reject) => {
      toast.promise<T>(
        async () => {
          return await fn();
        },
        {
          loading: promiseToast.loading,
          success: (data: T) => {
            resolve(data);
            return promiseToast.success;
          },
          error: (error: unknown) => {
            reject(error);

            if (isSessionExpiry(error)) {
              return SESSION_EXPIRED_MESSAGE;
            }

            // sonner renders this toast itself, so only the message is
            // swapped here. The resend button lives on the verification
            // banner, which is on screen whenever this can happen.
            if (isProblemType(error, ProblemType.EmailNotVerified)) {
              return UNVERIFIED_EMAIL_MESSAGE;
            }

            return deriveError(error);
          },
          finally: cleanup,
          action: action,
        },
      );
    });
  }

  try {
    return await fn();
  } catch (error) {
    await onError?.(error);

    // An unverified email surfaces as a plain 403; catching it centrally turns
    // every blocked action into a prompt with a way to fix it.
    if (isSessionExpiry(error)) {
      toast.info(SESSION_EXPIRED_MESSAGE, { id: "session-expired" });
    } else if (errorToast && !handleUnverifiedEmailError(error)) {
      toast.error(deriveError(error));
    }
  } finally {
    await cleanup?.();
  }

  return;
}
