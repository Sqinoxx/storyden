import { Account, Permission } from "@/api/openapi-schema";
import { hasPermission } from "@/utils/permissions";

export function canSeeMarketplace(
  enabled: boolean | undefined,
  session: Account | undefined,
) {
  return (
    enabled === true ||
    hasPermission(session, Permission.ADMINISTRATOR, Permission.MANAGE_LISTINGS)
  );
}

export function isMarketplaceBeta(enabled: boolean | undefined) {
  return enabled !== true;
}
