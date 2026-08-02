import { PropsWithChildren } from "react";

import { Box, HStack, styled } from "@/styled-system/jsx";
import { ImpressumLink } from "@/components/site/ImpressumModal/ImpressumLink";
import { CookieLink } from "@/components/site/CookieNotice/CookieLink";

export function ContextPane({ children }: PropsWithChildren) {
  return (
    <styled.nav
      display="flex"
      flexDir="column"
      alignItems="center"
      justifyContent="space-between"
      gap="2"
      width="full"
      height="full"
    >
      <Box
        id="desktop-nav-right"
        display="flex"
        flexDir="column"
        gap="3"
        w="full"
        height="full"
        minH="0"
        overflowY="auto"
      >
        {children}
      </Box>

      <HStack color="fg.subtle" fontSize="xs" flexShrink="0" py="1" gap="2">
        <ImpressumLink />
        <styled.span color="fg.subtle">•</styled.span>
        <CookieLink />
        <styled.span color="fg.subtle">•</styled.span>
        <p>powered by storyden</p>
      </HStack>
    </styled.nav>
  );
}

