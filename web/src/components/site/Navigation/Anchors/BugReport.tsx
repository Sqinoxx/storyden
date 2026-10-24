"use client";

import { BugIcon } from "@/components/ui/icons/Bug";
import { Item } from "@/components/ui/menu";
import { useTranslation } from "@/lib/i18n";

export const BugReportID = "bug-report";

export function BugReportMenuItem() {
  const t = useTranslation();
  return (
    <Item value={BugReportID}>
      <BugIcon />
      &nbsp;<span>{t.nav.reportBug}</span>
    </Item>
  );
}
