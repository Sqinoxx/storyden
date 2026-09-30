"use client";

import { CalendarIcon } from "@/components/ui/icons/Calendar";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps } from "./Anchor";

export const DailyLibraryID = "daily-library";
export const DailyLibraryRoute = "/admin/tagesarchiv";

export function DailyLibraryAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={DailyLibraryID}
      route={DailyLibraryRoute}
      label={t.nav.dailyLibrary}
      icon={<CalendarIcon />}
      {...props}
    />
  );
}
