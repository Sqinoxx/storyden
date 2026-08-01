import { flatten } from "lodash";

import { Account, AccountCommonProps, Permission } from "@/api/openapi-schema";

export function hasPermission(
  account?: Account | AccountCommonProps,
  ...permissions: Permission[]
) {
  if (!account) return false;

  // extract each permission from each role
  const accountPermissions = new Set(
    flatten(account.roles.map((role) => role.permissions)),
  );

  if (accountPermissions.has("ADMINISTRATOR")) {
    return true;
  }

  return permissions.some((permission) => accountPermissions.has(permission));
}

export function hasPermissionOr(
  account?: Account | AccountCommonProps,
  fn?: () => boolean,
  ...permissions: Permission[]
) {
  if (!account) return false;

  const hasPerm = hasPermission(account, ...permissions);
  if (hasPerm) {
    return true;
  }

  if (fn?.()) {
    return true;
  }

  return false;
}

export function isModeratorOrAdmin(account?: Account | AccountCommonProps) {
  return hasPermission(
    account,
    Permission.ADMINISTRATOR,
    Permission.MANAGE_POSTS,
    Permission.MANAGE_ROLES,
    Permission.MANAGE_LIBRARY,
    Permission.MANAGE_CATEGORIES,
    Permission.MANAGE_ACCOUNTS,
    Permission.MANAGE_SETTINGS,
    Permission.MANAGE_WARNINGS,
    Permission.MANAGE_SUSPENSIONS,
    Permission.MANAGE_REPORTS,
  );
}

