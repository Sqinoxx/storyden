import { WithDisclosure } from "@/utils/useDisclosure";

import { Button } from "@/components/ui/button";
import { HStack, VStack, styled } from "@/styled-system/jsx";

import { Props, useMemberSuspension } from "./useMemberSuspension";
import { useTranslation } from "@/lib/i18n";

export function MemberSuspensionConfirmation(props: WithDisclosure<Props>) {
  const t = useTranslation();
  const { handlers } = useMemberSuspension(props);

  return (
    <VStack alignItems="start">
      {props.profile.suspended ? (
        <styled.p>
          {t.moderation.confirmReinstate.replace("{name}", props.profile.name)}
        </styled.p>
      ) : (
        <styled.p>
          {t.moderation.confirmSuspend.replace("{name}", props.profile.name)}
        </styled.p>
      )}

      <HStack w="full">
        <Button type="button" flexGrow="1" onClick={props.onClose}>
          {t.common.cancel}
        </Button>

        {props.profile.suspended ? (
          <Button
            // w="full"
            flexGrow="1"
            colorPalette="red"
            onClick={handlers.handleReinstate}
          >
            {t.actions.unsuspend}
          </Button>
        ) : (
          <Button
            // w="full"
            flexGrow="1"
            colorPalette="red"
            onClick={handlers.handleSuspension}
          >
            {t.actions.suspend}
          </Button>
        )}
      </HStack>
    </VStack>
  );
}
