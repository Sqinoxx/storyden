"use client";

import { Lock } from "lucide-react";

import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useAccessPasses } from "@/lib/access/useAccess";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps } from "./Anchor";

export const AccessID = "access";
export const AccessRoute = "/access";

export function useHasAccessPasses() {
  const { passes } = useAccessPasses();
  return passes.length > 0;
}

export function AccessAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={AccessID}
      route={AccessRoute}
      label={t.nav.access}
      icon={<Lock size={16} />}
      {...props}
    />
  );
}
