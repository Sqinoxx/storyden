import { toast } from "sonner";

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

  return buildResult<T>(response);
};

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
    if (errorToast && !handleUnverifiedEmailError(error)) {
      toast.error(deriveError(error));
    }
  } finally {
    await cleanup?.();
  }

  return;
}
