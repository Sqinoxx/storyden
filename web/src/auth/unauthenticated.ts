import { RequestError } from "@/api/common";

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

export function matchesPage(pages: string[], pathName: string): boolean {
  return pages.some(
    (page) => pathName === page || pathName.startsWith(`${page}/`),
  );
}

export function isPublicPage(pathName: string): boolean {
  return matchesPage(PUBLIC_PAGES, pathName);
}

export function isUnauthenticatedError(error: unknown): boolean {
  return error instanceof RequestError && error.status === 401;
}
