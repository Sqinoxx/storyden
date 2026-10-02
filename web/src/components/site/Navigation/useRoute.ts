import { usePathname } from "next/navigation";

import { Translations, useTranslation } from "@/lib/i18n";

export type Route = {
  name: RouteName;
  label: string;
};

type RouteName = "index" | "library" | "drive" | "admin" | "settings";

function routeLabel(t: Translations, name: RouteName) {
  switch (name) {
    case "index":
      return t.nav.home;
    case "library":
      return t.nav.library;
    case "drive":
      return t.nav.drive;
    case "admin":
      return t.nav.admin;
    case "settings":
      return t.nav.settings;
  }
}

const mapping: Record<string, RouteName> = {
  "/": "index",
  "/d": "index",
  "/l": "library",
  "/drive": "drive",
  "/admin": "admin",
  "/settings": "settings",
};

function routeFromPrefix(t: Translations, prefix: string): Route | undefined {
  const routeName = mapping[prefix];
  if (!routeName) {
    return undefined;
  }
  return {
    name: routeName,
    label: routeLabel(t, routeName),
  };
}

export function useRoute(): Route | undefined {
  const t = useTranslation();
  const pathname = usePathname();
  if (pathname[0] !== "/") {
    throw new Error(
      `useRoute: Invalid pathname "${pathname}". Expected a path starting with "/".`,
    );
  }

  const parts = pathname.split("/");
  if (parts.length < 2) {
    console.warn("useRoute: unexpected pathname format", pathname);
    return routeFromPrefix(t, "/");
  }

  const first = parts[1];

  const prefix = `/${first}`;

  const route = routeFromPrefix(t, prefix);

  return route;
}
