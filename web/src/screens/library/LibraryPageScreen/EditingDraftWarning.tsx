"use client";

import { Heading } from "@/components/ui/heading";
import { LStack, WStack, styled } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";

import { useLibraryPagePermissions } from "./permissions";

export function EditingDraftWarning() {
  const { isAllowedToDirectEdit } = useLibraryPagePermissions();
  const t = useTranslation();

  const label = isAllowedToDirectEdit
    ? t.library.editingDraftApplied
    : t.library.editingDraftApproved;

  return (
    <LStack
      borderWidth="thin"
      borderStyle="dashed"
      borderColor="visibility.draft.border"
      borderRadius="sm"
      bgColor="bg.subtle"
      p="2"
      gap="0"
    >
      <Heading size="sm">{t.library.editingDraftTitle}</Heading>
      <styled.span color="fg.muted" fontSize="sm">
        {label}
      </styled.span>
    </LStack>
  );
}
