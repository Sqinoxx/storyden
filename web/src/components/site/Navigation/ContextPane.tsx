import { PropsWithChildren } from "react";

import { Box, HStack, styled } from "@/styled-system/jsx";

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

      <HStack color="fg.subtle" fontSize="xs" flexShrink="0" py="1">
        {/* TODO: Provide links to privacy/terms/etc custom pages */}
        {/* <p>copyright {settings.owner}</p> */}
        {/* <a href={PrivacyRoute}>privacy</a> */}
        <p>powered by storyden</p>
      </HStack>
    </styled.nav>
  );
}

