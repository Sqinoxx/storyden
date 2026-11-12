"use client";

import { FieldValues, Path, PathValue, useFormContext } from "react-hook-form";

import { Asset } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { CloseIcon } from "@/components/ui/icons/Close";
import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { IconButton } from "@/components/ui/icon-button";
import { useTranslation } from "@/lib/i18n";
import { formatTermLabel } from "@/lib/thread/semester";
import { HStack, styled } from "@/styled-system/jsx";

import { useSemesterSuggestion } from "./useSemesterSuggestion";

type Props<T extends FieldValues> = {
  name: Path<T>;
  attachments: Asset[];
};

export function SemesterSuggestionHint<T extends FieldValues>({
  name,
  attachments,
}: Props<T>) {
  const t = useTranslation();
  const { control, setValue } = useFormContext<T>();
  const { suggestion, dismiss } = useSemesterSuggestion({
    control,
    name,
    attachments,
  });

  if (!suggestion) return null;

  const apply = () =>
    setValue(name, suggestion.key as PathValue<T, Path<T>>, {
      shouldDirty: true,
    });

  return (
    <HStack
      w="full"
      mt="3"
      gap="2"
      px="3"
      py="1"
      borderRadius="md"
      bg="bg.muted"
      color="fg.muted"
      fontSize="sm"
      flexWrap="wrap"
    >
      <IntelligenceIcon width="4" height="4" flexShrink="0" />
      <styled.span flex="1" minW="0">
        {t.thread.semesterSuggestion.replace(
          "{term}",
          formatTermLabel(suggestion.term),
        )}
        {suggestion.evidence && (
          <styled.span color="fg.subtle">
            {" "}
            {t.thread.semesterSuggestionEvidence.replace(
              "{evidence}",
              suggestion.evidence,
            )}
          </styled.span>
        )}
      </styled.span>
      <Button type="button" size="xs" variant="subtle" onClick={apply}>
        {t.thread.semesterSuggestionApply}
      </Button>
      <IconButton
        type="button"
        size="xs"
        variant="ghost"
        aria-label={t.thread.semesterSuggestionDismiss}
        title={t.thread.semesterSuggestionDismiss}
        onClick={dismiss}
      >
        <CloseIcon />
      </IconButton>
    </HStack>
  );
}
