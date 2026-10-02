import { PropsWithChildren } from "react";

import { LinkButton } from "@/components/ui/link-button";
import { Trans } from "@/lib/i18n";
import { allowsPublicRegistration } from "@/lib/settings/registration";
import { getSettings } from "@/lib/settings/settings-server";
import { HStack, VStack } from "@/styled-system/jsx";

export default async function Layout({ children }: PropsWithChildren) {
  const { registration_mode } = await getSettings();
  const canRegister = allowsPublicRegistration(registration_mode);

  return (
    <VStack w="full">
      {children}

      <HStack>
        <LinkButton size="xs" variant="ghost" href="/login">
          <Trans path="auth.login" />
        </LinkButton>

        {canRegister && (
          <LinkButton size="xs" variant="subtle" href="/register">
            <Trans path="auth.register" />
          </LinkButton>
        )}
      </HStack>
    </VStack>
  );
}
