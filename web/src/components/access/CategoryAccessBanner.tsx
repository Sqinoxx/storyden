"use client";

import { LockOpen } from "lucide-react";

import { useCategoryAccess } from "@/lib/access/useAccess";
import { useLanguage } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

import { AccessPassCard } from "./AccessPassCard";

type Props = {
  categoryID: string;
};

export function CategoryAccessBanner({ categoryID }: Props) {
  const { t, language } = useLanguage();
  const access = useCategoryAccess(categoryID);

  if (access.state === "open") return null;

  if (access.state === "unlocked") {
    const expires = access.held.membership?.expires_at;
    const label = expires
      ? t.access.activeUntil.replace(
          "{date}",
          new Date(expires).toLocaleDateString(language),
        )
      : t.access.activeForever;

    return (
      <HStack
        gap="2"
        fontSize="sm"
        color="fg.muted"
        data-testid="access-unlocked-hint"
      >
        <LockOpen size={14} />
        <styled.span>
          {access.held.name} · {label}
        </styled.span>
      </HStack>
    );
  }

  return (
    <LStack gap="3" w="full">
      {access.passes.map((p) => (
        <AccessPassCard key={p.id} pass={p} />
      ))}
    </LStack>
  );
}
