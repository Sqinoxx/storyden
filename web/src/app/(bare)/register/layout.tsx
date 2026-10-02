import { PropsWithChildren } from "react";

import { LinkButton } from "@/components/ui/link-button";
import { Trans } from "@/lib/i18n";
import { VStack } from "@/styled-system/jsx";

export default async function Layout({ children }: PropsWithChildren) {
  return (
    <VStack w="full">
      {children}

      <LinkButton size="xs" variant="subtle" href="/login">
        <Trans path="auth.login" />
      </LinkButton>
    </VStack>
  );
}
