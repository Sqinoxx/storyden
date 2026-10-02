import React, { PropsWithChildren } from "react";

import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useDisclosure } from "@/utils/useDisclosure";

import { Button } from "@/components/ui/button";

import { MemberSuspensionConfirmation } from "./MemberSuspensionConfirmation";
import { Props } from "./useMemberSuspension";
import { useTranslation } from "@/lib/i18n";

export function MemberSuspensionTrigger({
  children,
  profile,
}: PropsWithChildren<Props>) {
  const t = useTranslation();
  const { onOpen, isOpen, onClose } = useDisclosure();

  const title = profile.suspended
    ? t.moderation.reinstateAccount.replace("{name}", profile.name)
    : t.moderation.suspendAccount.replace("{name}", profile.name);

  return (
    <>
      {children ? (
        React.cloneElement(
          // not sure why types broken here, but it works fine.
          children as any,
          {
            onClick: onOpen,
          },
        )
      ) : (
        <Button colorPalette="red" onClick={onOpen}>
          {profile.suspended ? t.actions.unsuspend : t.actions.suspend}
        </Button>
      )}

      <ModalDrawer isOpen={isOpen} onClose={onClose} title={title}>
        <MemberSuspensionConfirmation onClose={onClose} profile={profile} />
      </ModalDrawer>
    </>
  );
}
