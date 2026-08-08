"use client";

import { Folder } from "lucide-react";

import { useDriveFolderList } from "@/api/openapi-client/drive";
import { LinkButtonStyleProps } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";

import { Anchor, AnchorProps, MenuItem } from "./Anchor";

export const DriveID = "drive";
export const DriveRoute = "/drive";
export const DriveLabel = "Drive";

/**
 * Hides the entry unless there is something behind it. The folder list already
 * accounts for the caller's visibility, so this answers "is there anything here
 * for you" rather than "is the integration switched on", which is what a member
 * actually cares about.
 */
export function useHasDriveFolders() {
  const { data } = useDriveFolderList();

  return (data?.folders.length ?? 0) > 0;
}

export function DriveAnchor(props: AnchorProps & LinkButtonStyleProps) {
  const t = useTranslation();
  return (
    <Anchor
      id={DriveID}
      route={DriveRoute}
      label={t.nav.drive}
      icon={<Folder size={16} />}
      {...props}
    />
  );
}

export function DriveMenuItem() {
  const t = useTranslation();
  return (
    <MenuItem
      id={DriveID}
      route={DriveRoute}
      label={t.nav.drive}
      icon={<Folder size={16} />}
    />
  );
}
