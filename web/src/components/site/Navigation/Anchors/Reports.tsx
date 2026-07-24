"use client";

import { ReportIcon } from "@/components/ui/icons/Report";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";
import { useTranslation } from "@/lib/i18n";

export const ReportsID = "reports";
export const ReportsRoute = "/reports";
export const ReportsLabel = "Reports";

export function ReportsAnchor(props: AnchorProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={ReportsID}
      route={ReportsRoute}
      label={t.nav.reports}
      icon={<ReportIcon />}
      {...props}
    />
  );
}

export function ReportsMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={ReportsID}
      route={ReportsRoute}
      label={t.nav.reports}
      icon={<ReportIcon />}
    />
  );
}
