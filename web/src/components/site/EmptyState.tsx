"use client";

import { JSX, PropsWithChildren } from "react";

import { useSession } from "@/auth";
import { useTranslation } from "@/lib/i18n";
import { Center, VStack, VstackProps } from "@/styled-system/jsx";
import { vstack } from "@/styled-system/patterns";

import { EmptyIcon } from "../ui/icons/Empty";

type Props = {
  icon?: JSX.Element;
  unauthenticatedLabel?: string;
  authenticatedLabel?: string;
  hideContributionLabel?: boolean;
};

export function EmptyState({
  icon,
  unauthenticatedLabel,
  authenticatedLabel,
  hideContributionLabel,
  children,
  ...props
}: PropsWithChildren<Props & VstackProps>) {
  const t = useTranslation();
  const session = useSession();

  const contributionLabel = session
    ? (authenticatedLabel ?? "")
    : (unauthenticatedLabel ?? "");

  return (
    <Center className={vstack(props)} p="8" gap="2" color="fg.subtle">
      {icon || <EmptyIcon />}

      <VStack gap="1" textAlign="center" fontStyle="italic">
        {children || <p>{t.common.noContent}</p>}
        {!hideContributionLabel && <p>{contributionLabel}</p>}
      </VStack>
    </Center>
  );
}
