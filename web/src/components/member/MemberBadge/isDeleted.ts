import { ProfileReference } from "@/api/openapi-schema";

export function isDeletedAccount(
  profile?:
    | (Partial<ProfileReference> & { deletedAt?: string | null })
    | null,
): boolean {
  if (!profile) return false;
  return Boolean(
    profile.handle?.startsWith("deleted-") ||
      profile.name === "Deleted User" ||
      profile.deletedAt,
  );
}

