import type {
  AccessGrant,
  AccessGrantStatus,
  AccessPass,
  Category,
} from "@/api/openapi-schema";

type CategoryLike = Pick<Category, "id" | "parent">;

export function categoryAncestry(
  categories: CategoryLike[],
  categoryID: string,
): string[] {
  const byID = new Map(categories.map((c) => [c.id, c]));
  const chain: string[] = [];
  const seen = new Set<string>();

  let cursor: string | undefined = categoryID;
  while (cursor && !seen.has(cursor)) {
    seen.add(cursor);
    chain.push(cursor);
    cursor = byID.get(cursor)?.parent;
  }

  return chain;
}

// Passes attached to the category itself or any of its ancestors, mirroring
// how the backend inherits locks down the category tree.
export function passesCoveringCategory(
  passes: AccessPass[],
  categories: CategoryLike[],
  categoryID: string,
): AccessPass[] {
  const chain = new Set(categoryAncestry(categories, categoryID));
  return passes.filter((p) => p.category_ids.some((id) => chain.has(id)));
}

export function isPassHeld(pass: AccessPass): boolean {
  return pass.membership?.status === "active";
}

export type CategoryAccess =
  | { state: "open" }
  | { state: "locked"; passes: AccessPass[] }
  | { state: "unlocked"; passes: AccessPass[]; held: AccessPass };

export function categoryAccess(
  passes: AccessPass[],
  categories: CategoryLike[],
  categoryID: string,
  bypass: boolean,
): CategoryAccess {
  const covering = passesCoveringCategory(passes, categories, categoryID);
  if (covering.length === 0) {
    return { state: "open" };
  }

  const held = covering.find(isPassHeld);
  if (held) {
    return { state: "unlocked", passes: covering, held };
  }

  if (bypass) {
    return { state: "open" };
  }

  return { state: "locked", passes: covering };
}

export function lockedCategoryIDs(
  passes: AccessPass[],
  categories: CategoryLike[],
  bypass: boolean,
): Set<string> {
  const out = new Set<string>();
  if (bypass || passes.length === 0) return out;

  for (const c of categories) {
    if (categoryAccess(passes, categories, c.id, false).state === "locked") {
      out.add(c.id);
    }
  }

  return out;
}

export type MembershipAction = "login" | "request" | "withdraw" | "none";

export function membershipAction(
  pass: AccessPass,
  loggedIn: boolean,
): MembershipAction {
  if (!loggedIn) return "login";

  const status: AccessGrantStatus | undefined = pass.membership?.status;
  switch (status) {
    case "active":
      return "none";
    case "requested":
      return "withdraw";
    default:
      return pass.requestable ? "request" : "none";
  }
}

export function matchesGrantQuery(grant: AccessGrant, query: string): boolean {
  const q = query.trim().toLowerCase().replace(/^@/, "");
  if (!q) return true;

  return [grant.account.name, grant.account.handle, grant.reference_code].some(
    (v) => v.toLowerCase().includes(q),
  );
}

export const EXPIRING_SOON_DAYS = 14;

export type PassStats = {
  active: number;
  requested: number;
  expiring: number;
  expired: number;
};

export function passStats(
  grants: AccessGrant[],
  now: Date,
): Map<string, PassStats> {
  const soon = now.getTime() + EXPIRING_SOON_DAYS * 24 * 60 * 60 * 1000;
  const out = new Map<string, PassStats>();

  for (const g of grants) {
    const s = out.get(g.pass.id) ?? {
      active: 0,
      requested: 0,
      expiring: 0,
      expired: 0,
    };

    switch (g.status) {
      case "active":
        s.active++;
        if (g.expires_at && new Date(g.expires_at).getTime() <= soon) {
          s.expiring++;
        }
        break;
      case "requested":
        s.requested++;
        break;
      case "expired":
        s.expired++;
        break;
    }

    out.set(g.pass.id, s);
  }

  return out;
}
