"use client";

import { useRef, useState } from "react";

import { invitationCreate } from "@/api/openapi-client/invitations";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { Button } from "@/components/ui/button";
import * as Clipboard from "@/components/ui/clipboard";
import { IconButton } from "@/components/ui/icon-button";
import { CheckIcon } from "@/components/ui/icons/Check";
import { CopyIcon } from "@/components/ui/icons/Copy";
import { InvitationIcon } from "@/components/ui/icons/Invitation";
import { Input } from "@/components/ui/input";
import { Item } from "@/components/ui/menu";
import { createInvitationLink } from "@/lib/invitation/link";
import { Center, LStack, VStack, WStack, styled } from "@/styled-system/jsx";
import { deriveError } from "@/utils/error";
import { useTranslation } from "@/lib/i18n";

type InvitationState =
  | { status: "idle" | "loading" }
  | { status: "success"; link: string }
  | { status: "error"; message: string };

export const InvitationID = "invitation";
export const InvitationLabel = "Invite";

export function useInvitation() {
  const [invitation, setInvitation] = useState<InvitationState>({
    status: "idle",
  });
  const isCreating = useRef(false);

  async function createInvitation() {
    if (isCreating.current) {
      return;
    }

    isCreating.current = true;
    setInvitation({ status: "loading" });

    try {
      const created = await invitationCreate({});
      setInvitation({
        status: "success",
        link: createInvitationLink(created.id),
      });
    } catch (error) {
      setInvitation({
        status: "error",
        message: deriveError(error),
      });
    } finally {
      isCreating.current = false;
    }
  }

  return { invitation, createInvitation };
}

export function InvitationMenuItem() {
  const t = useTranslation();
  return (
    <Item value={InvitationID}>
      <InvitationIcon />
      &nbsp;<span>{t.actions.invite}</span>
    </Item>
  );
}

export function InvitationModal({
  invitation,
  isOpen,
  onRetry,
  onClose,
}: {
  invitation: InvitationState;
  isOpen?: boolean;
  onRetry: () => void;
  onClose: () => void;
}) {
  const t = useTranslation();
  return (
    <ModalDrawer title={t.actions.inviteSomeone} isOpen={isOpen} onClose={onClose}>
      <InvitationModalContent
        invitation={invitation}
        onRetry={onRetry}
        onClose={onClose}
      />
    </ModalDrawer>
  );
}

function InvitationModalContent({
  invitation,
  onRetry,
  onClose,
}: {
  invitation: InvitationState;
  onRetry: () => void;
  onClose: () => void;
}) {
  const t = useTranslation();

  if (invitation.status === "success") {
    return (
      <LStack gap="6">
        <LStack gap="2">
          <styled.p color="fg.muted">
            Send this invitation link to someone you would like to welcome into
            the community.
          </styled.p>
        </LStack>

        <Clipboard.Root w="full" value={invitation.link}>
          <Clipboard.Control gap="0">
            <Clipboard.Input asChild>
              <Input readOnly aria-label="Invitation link" />
            </Clipboard.Input>
            <Clipboard.Trigger asChild>
              <IconButton variant="outline" aria-label="Copy invitation link">
                <Clipboard.Indicator copied={<CheckIcon />}>
                  <CopyIcon />
                </Clipboard.Indicator>
              </IconButton>
            </Clipboard.Trigger>
          </Clipboard.Control>
        </Clipboard.Root>

        <WStack>
          <Button w="full" onClick={onClose}>
            {t.actions.done}
          </Button>
        </WStack>
      </LStack>
    );
  }

  if (invitation.status === "error") {
    return (
      <LStack gap="6">
        <LStack gap="2">
          <styled.p color="fg.muted">
            Something went wrong while creating the invitation.
          </styled.p>
          <styled.p color="fg.error" fontSize="sm">
            {invitation.message}
          </styled.p>
        </LStack>

        <WStack>
          <Button w="full" variant="outline" onClick={onClose}>
            {t.actions.close}
          </Button>
          <Button w="full" onClick={onRetry}>
            {t.actions.tryAgain}
          </Button>
        </WStack>
      </LStack>
    );
  }

  return (
    <Center minH="36">
      <VStack gap="3" textAlign="center">
        <IconButton variant="ghost" loading aria-label="Creating invitation" />
        <styled.p color="fg.muted">Creating an invitation...</styled.p>
      </VStack>
    </Center>
  );
}
