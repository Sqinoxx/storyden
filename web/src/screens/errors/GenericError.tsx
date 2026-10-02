"use client";

import { usePathname } from "next/navigation";

import { UnreadyBanner } from "@/components/site/Unready";

import { Button } from "@/components/ui/button";
import { LinkButton } from "@/components/ui/link-button";
import { HStack, VStack } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";

export function GenericError({
  reset,
  message,
}: {
  reset?: () => void;
  message?: string;
}) {
  const t = useTranslation();
  const pathName = usePathname();

  const isHome = pathName === "/";

  return (
    <VStack p="4" h="dvh" justify="center">
      <VStack maxW="sm" minH="60" gap="8">
        <UnreadyBanner error={message ?? t.validation.unexpectedError} />
        <HStack>
          {!isHome && (
            <LinkButton variant="subtle" href="/">
              {t.nav.home}
            </LinkButton>
          )}
          <Button variant="outline" onClick={reset}>
            {t.common.retry}
          </Button>
        </HStack>
      </VStack>
    </VStack>
  );
}
