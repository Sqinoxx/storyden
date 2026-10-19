import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { useAccountGet } from "@/api/openapi-client/accounts";

import { usePublicRegistration } from "@/lib/settings/registration";

import {
  isPublicPage,
  isUnauthenticatedError,
  matchesPage,
} from "./unauthenticated";

const PRIVATE_PAGES = ["/settings", "/new", "/admin"];

export function useAuthProvider() {
  const { isLoading, data, error } = useAccountGet();
  const { push, replace } = useRouter();
  const pathname = usePathname();
  const canRegister = usePublicRegistration();

  const loggedIn = Boolean(data) && !error;
  const isPrivate = pathname && matchesPage(PRIVATE_PAGES, pathname);
  const isPublic = pathname && isPublicPage(pathname);

  useEffect(() => {
    if (isLoading) return;

    if (isUnauthenticatedError(error) && !isPublic) {
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
    isPublic,
    pathname,
    push,
    replace,
  ]);

  return;
}
