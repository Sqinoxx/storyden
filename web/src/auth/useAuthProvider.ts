import { usePathname, useRouter } from "next/navigation";
import { useEffect } from "react";

import { RequestError } from "@/api/common";
import { useAccountGet } from "@/api/openapi-client/accounts";

import { usePublicRegistration } from "@/lib/settings/registration";

const PRIVATE_PAGES = ["/settings", "/new", "/admin"];

const PUBLIC_PAGES = [
  "/login",
  "/register",
  "/password-reset",
  "/auth",
  "/invitation",
  "/oauth",
  "/logout",
  "/impressum",
];

function matchesPage(pages: string[], pathName: string): boolean {
  return pages.some(
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

  const loggedIn = Boolean(data) && !error;
  const isPrivate = pathname && matchesPage(PRIVATE_PAGES, pathname);
  const isPublic = pathname && matchesPage(PUBLIC_PAGES, pathname);

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
