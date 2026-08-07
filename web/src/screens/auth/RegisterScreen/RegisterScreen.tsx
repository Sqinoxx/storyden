import { AuthMode, RegistrationMode } from "@/api/openapi-schema";
import { authProviderList } from "@/api/openapi-server/auth";
import * as Alert from "@/components/ui/alert";
import { WarningIcon } from "@/components/ui/icons/Warning";
import { VStack, styled } from "@/styled-system/jsx";

import { RegisterEmailForm } from "./RegisterEmail/RegisterEmailForm";
import { RegisterHandleForm } from "./RegisterHandle/RegisterHandleForm";
import { RegisterPhoneForm } from "./RegisterPhone/RegisterPhoneForm";

type Props = {
  invitationID?: string;
  registrationMode: RegistrationMode;
};

export async function RegisterScreen({
  invitationID,
  registrationMode,
}: Props) {
  const { data } = await authProviderList({
    cache: "no-store",
  });

  const isInviteOnly = registrationMode === RegistrationMode.invitation;
  if (isInviteOnly && !invitationID) {
    return (
      <VStack textAlign="center">
        <styled.h1 fontWeight="bold">Registration is invite-only.</styled.h1>
        <styled.p color="fg.muted" textWrap="balance">
          Ask a community member or administrator for an invitation link to
          join.
        </styled.p>
      </VStack>
    );
  }

  const isDisabled = registrationMode === RegistrationMode.disabled;
  if (isDisabled) {
    return (
      <VStack textAlign="center">
        <styled.h1 fontWeight="bold">
          Registration is currently closed.
        </styled.h1>
        <styled.p color="fg.muted" textWrap="balance">
          This site has closed public registration of accounts.
        </styled.p>
      </VStack>
    );
  }

  const form = getForm(data.mode, invitationID);
  if (!form) {
    console.error("no authentication modes available");

    return (
      <VStack>
        <p>This instance is private.</p>
      </VStack>
    );
  }

  return (
    <VStack w="full" gap="4">
      <Alert.Root>
        <Alert.Icon asChild>
          <WarningIcon />
        </Alert.Icon>
        <Alert.Content>
          <Alert.Title>Bleib anonym</Alert.Title>
          <Alert.Description>
            Verwende bitte keinen Klarnamen und keine Uni-E-Mail-Adresse als
            Benutzername oder Kontakt. Wähle nichts, das auf deine echte
            Identität schließen lässt.
          </Alert.Description>
        </Alert.Content>
      </Alert.Root>

      {form}
    </VStack>
  );
}

function getForm(mode: AuthMode | undefined, invitationID: string | undefined) {
  switch (mode) {
    case AuthMode.handle:
      return (
        <RegisterHandleForm webauthn={false} invitationID={invitationID} />
      );

    case AuthMode.email:
      return <RegisterEmailForm invitationID={invitationID} />;

    case AuthMode.phone:
      return <RegisterPhoneForm />;

    default:
      return null;
  }
}
