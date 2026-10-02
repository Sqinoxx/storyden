import { UnreadyBanner } from "@/components/site/Unready";

import { LinkButton } from "@/components/ui/link-button";
import { Trans } from "@/lib/i18n";
import { VStack } from "@/styled-system/jsx";

export function ThreadNotFoundError() {
  return (
    <VStack p="4" h="dvh" justify="center">
      <VStack maxW="sm" minH="60" gap="8">
        <UnreadyBanner error="The link to this thread did not lead anywhere." />
        <LinkButton variant="subtle" href="/">
          <Trans path="nav.home" />
        </LinkButton>
      </VStack>
    </VStack>
  );
}
