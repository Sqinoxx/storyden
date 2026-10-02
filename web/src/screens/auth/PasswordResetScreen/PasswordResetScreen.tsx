import { AuthMode } from "@/api/openapi-schema";
import { authProviderList } from "@/api/openapi-server/auth";
import { Unready } from "@/components/site/Unready";
import { Trans } from "@/lib/i18n";
import { VStack } from "@/styled-system/jsx";

import { PasswordResetEmailScreen } from "./PasswordResetEmailScreen";

export async function PasswordResetScreen() {
  const { data } = await authProviderList({
    cache: "no-store",
  });

  switch (data.mode) {
    case AuthMode.handle:
      return <Warning />;

    case AuthMode.email:
      return <PasswordResetEmailScreen />;

    case AuthMode.phone:
      return <Warning />;

    default:
      return <Unready error="No authentication modes are currently enabled." />;
  }
}

function Warning() {
  return (
    <VStack textWrap="balance" textAlign="center">
      <p>
        <Trans path="auth.passwordResetDisabled" />
      </p>
    </VStack>
  );
}
