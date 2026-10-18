import { usePathname, useRouter } from "next/navigation";
import { useEffect, useRef } from "react";

import { RequestError } from "@/api/common";
import { useAccountGet } from "@/api/openapi-client/accounts";

import { usePublicRegistration } from "@/lib/settings/registration";

const PRIVATE_PAGES = ["/settings", "/new", "/admin"];

function privatePage(pathName: string): boolean {
  return PRIVATE_PAGES.some(
    (page) => pathName === page || pathName.startsWith(`${page}/`),
  );
}

export function isUnauthenticatedError(error: unknown): boolean {
  return error instanceof RequestError && error.status === 401;
}

export function useAuthProvider() {
  const { isLoading, data, error } = useAccountGet();
  const { push, replace } = useRouter();
  const pathname = usePathname();
  const canRegister = usePublicRegistration();
  const wasLoggedIn = useRef(false);

  const loggedIn = Boolean(data) && !error;
  const isPrivate = pathname && privatePage(pathname);

  useEffect(() => {
    if (isLoading) return;

    if (loggedIn) {
      wasLoggedIn.current = true;
    } else if (wasLoggedIn.current && isUnauthenticatedError(error)) {
      wasLoggedIn.current = false;
      replace("/login");
      return;
    }

    if (!loggedIn && isPrivate) {
      push(canRegister ? "/register" : "/login");
    }
    if (loggedIn && (pathname === "/register" || pathname === "/login")) {
      push("/");
    }
  }, [
    canRegister,
    isLoading,
    loggedIn,
    error,
    isPrivate,
    pathname,
    push,
    replace,
  ]);

  return;
}
