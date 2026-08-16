"use client";

import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { HStack } from "@/styled-system/jsx";

export type Props = {
  canRegister: boolean;
};

export function LoginFooterLinks({ canRegister }: Props) {
  const t = useTranslation();

  return (
    <HStack>
      <LinkButton size="xs" variant="ghost" href="/password-reset">
        {t.auth.forgotPassword}
      </LinkButton>

      {canRegister && (
        <LinkButton size="xs" variant="subtle" href="/register">
          {t.auth.register}
        </LinkButton>
      )}
    </HStack>
  );
}
