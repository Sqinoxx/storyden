import { PropsWithChildren } from "react";

import { allowsPublicRegistration } from "@/lib/settings/registration";
import { getSettings } from "@/lib/settings/settings-server";
import { VStack } from "@/styled-system/jsx";

import { LoginFooterLinks } from "@/screens/auth/LoginScreen/LoginFooterLinks";

export default async function Layout({ children }: PropsWithChildren) {
  const { registration_mode } = await getSettings();
  const canRegister = allowsPublicRegistration(registration_mode);

  return (
    <VStack w="full">
      {children}

      <LoginFooterLinks canRegister={canRegister} />
    </VStack>
  );
}
