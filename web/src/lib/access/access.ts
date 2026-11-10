import type {
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
