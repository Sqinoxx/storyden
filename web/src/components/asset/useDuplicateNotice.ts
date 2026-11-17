"use client";

import { toast } from "sonner";

import { Asset, AssetDuplicateLocation } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";

function locationPath(l: AssetDuplicateLocation) {
  return l.kind === "thread" ? `/t/${l.slug}` : `/l/${l.slug}`;
}

export function useDuplicateNotice() {
  const t = useTranslation();

  return function notifyDuplicate(asset: Asset, name: string) {
    const dup = asset.duplicate_of;
    if (!dup) {
      return;
    }

    const first = dup.locations[0];
    const where = dup.locations.map((l) => `„${l.title}“`).join(", ");

    toast.info(t.upload.duplicateTitle.replace("{name}", name), {
      id: `duplicate-${asset.id}`,
      duration: 10000,
      description: where
        ? t.upload.duplicateIn.replace("{locations}", where)
        : t.upload.duplicateUnlisted,
      action: first
        ? {
            label: t.upload.duplicateOpen,
            onClick: () =>
              window.open(locationPath(first), "_blank", "noopener"),
          }
        : undefined,
    });
  };
}
