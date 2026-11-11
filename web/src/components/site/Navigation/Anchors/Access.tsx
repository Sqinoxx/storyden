"use client";

import { Lock } from "lucide-react";

import { Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { useOpenRequests } from "@/components/access/manage/RequestsTab";
import { Badge } from "@/components/ui/badge";
import { LinkButton, LinkButtonStyleProps } from "@/components/ui/link-button";
import { useAccessPasses } from "@/lib/access/useAccess";
import { useTranslation } from "@/lib/i18n";
import { hasPermission } from "@/utils/permissions";

import { AnchorProps } from "./Anchor";

export const AccessID = "access";
export const AccessRoute = "/access";

export function useHasAccessPasses() {
  const session = useSession();
  const { passes } = useAccessPasses();
  return passes.length > 0 || hasPermission(session, Permission.MANAGE_ACCESS);
}

// Managers see the number of members waiting for their payment to be
// confirmed, so open requests do not sit unnoticed.
export function AccessAnchor({
  hideLabel,
  ...props
}: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  const session = useSession();
  const canManage = hasPermission(session, Permission.MANAGE_ACCESS);
  const { data } = useOpenRequests(canManage);
  const open = data?.grants.length ?? 0;

  return (
    <LinkButton href={AccessRoute} size="xs" p="1" variant="ghost" {...props}>
      <Lock size={16} width="1.5rem" />
      {!hideLabel && (
        <>
          &nbsp;<span>{t.nav.access}</span>
        </>
      )}
      {open > 0 && (
        <Badge
          size="sm"
          variant="solid"
          aria-label={`${open} ${t.access.manage.tabRequests}`}
        >
          {open}
        </Badge>
      )}
    </LinkButton>
  );
}
