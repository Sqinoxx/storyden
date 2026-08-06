"use client";

import { parseAsBoolean, useQueryState } from "nuqs";

import { Button } from "@/components/ui/button";
import { DeletedMemberIcon } from "@/components/ui/icons/DeletedMember";
import { useTranslation } from "@/lib/i18n";
import { HStack } from "@/styled-system/jsx";

type Props = {
  size?: "sm" | "md" | "lg";
};

export function HideDeletedFilter({ size = "md" }: Props) {
  const t = useTranslation();
  const [hideDeleted, setHideDeleted] = useQueryState(
    "hide_deleted",
    parseAsBoolean.withDefault(true),
  );

  const toggleHideDeleted = async () => {
    await setHideDeleted(hideDeleted ? false : null);
  };

  return (
    <Button
      variant={hideDeleted ? "outline" : "subtle"}
      size={size}
      flexShrink="0"
      whiteSpace="nowrap"
      onClick={toggleHideDeleted}
      aria-label={hideDeleted ? t.members.showDeleted : t.members.hideDeleted}
    >
      <HStack gap="1.5" whiteSpace="nowrap">
        <DeletedMemberIcon style={{ opacity: hideDeleted ? 0.4 : 1 }} />
        <span>
          {hideDeleted ? t.members.showDeleted : t.members.hideDeleted}
        </span>
      </HStack>
    </Button>
  );
}
