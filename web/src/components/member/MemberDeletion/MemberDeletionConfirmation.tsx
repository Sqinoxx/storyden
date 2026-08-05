import { WithDisclosure } from "@/utils/useDisclosure";
import { useTranslation } from "@/lib/i18n";

import { Button } from "@/components/ui/button";
import { HStack, VStack, styled } from "@/styled-system/jsx";

import { Props, useMemberDeletion } from "./useMemberDeletion";

export function MemberDeletionConfirmation(props: WithDisclosure<Props>) {
  const { handlers } = useMemberDeletion(props);
  const t = useTranslation();

  return (
    <VStack alignItems="start" gap="4">
      <styled.p>
        Are you sure you want to permanently delete the account <strong>{props.profile.name}</strong>? This action cannot be undone.
      </styled.p>

      <HStack w="full">
        <Button type="button" flexGrow="1" onClick={props.onClose}>
          {t.actions.close}
        </Button>

        <Button
          flexGrow="1"
          colorPalette="red"
          onClick={handlers.handleDelete}
        >
          {t.actions.deleteMember}
        </Button>
      </HStack>
    </VStack>
  );
}
