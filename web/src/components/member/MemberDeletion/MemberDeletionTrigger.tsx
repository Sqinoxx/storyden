import React, { PropsWithChildren } from "react";

import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useDisclosure } from "@/utils/useDisclosure";

import { Button } from "@/components/ui/button";

import { MemberDeletionConfirmation } from "./MemberDeletionConfirmation";
import { Props } from "./useMemberDeletion";
import { useTranslation } from "@/lib/i18n";

export function MemberDeletionTrigger({
  children,
  profile,
}: PropsWithChildren<Props>) {
  const t = useTranslation();
  const { onOpen, isOpen, onClose } = useDisclosure();

  const title = t.moderation.deleteAccount.replace("{name}", profile.name);

  return (
    <>
      {children ? (
        React.cloneElement(
          children as any,
          {
            onClick: onOpen,
          },
        )
      ) : (
        <Button colorPalette="red" onClick={onOpen}>
          {t.actions.deleteMember}
        </Button>
      )}

      <ModalDrawer isOpen={isOpen} onClose={onClose} title={title}>
        <MemberDeletionConfirmation onClose={onClose} profile={profile} />
      </ModalDrawer>
    </>
  );
}
