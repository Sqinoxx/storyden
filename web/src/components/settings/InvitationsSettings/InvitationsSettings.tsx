"use client";

import { formatDate } from "date-fns";

import { Invitation } from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { useConfirmation } from "@/components/site/useConfirmation";
import { Button } from "@/components/ui/button";
import * as Clipboard from "@/components/ui/clipboard";
import { Heading } from "@/components/ui/heading";
import { IconButton } from "@/components/ui/icon-button";
import { AddIcon } from "@/components/ui/icons/Add";
import { CheckIcon } from "@/components/ui/icons/Check";
import { CopyIcon } from "@/components/ui/icons/Copy";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { createInvitationLink } from "@/lib/invitation/link";
import { CardBox, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { MultiUseInvitationForm } from "./MultiUseInvitationForm";
import { useInvitationsSettings } from "./useInvitationsSettings";

export function InvitationsSettings() {
  const t = useTranslation();
  const state = useInvitationsSettings();

  if (!state.ready) {
    return <Unready error={state.error} />;
  }

  const {
    invitations,
    createdID,
    isCreating,
    isMultiFormOpen,
    handleCreate,
    handleDelete,
    handleDismissCreated,
    handleOpenMultiForm,
    handleCloseMultiForm,
    handleMultiCreated,
  } = state;

  return (
    <CardBox className={lstack()} gap="8">
      <LStack>
        <Heading size="md">{t.invitations.title}</Heading>
        <p>{t.invitations.description}</p>
      </LStack>

      <LStack>
        <WStack alignItems="center" color="fg.muted">
          <styled.p>
            {invitations.length === 1
              ? t.invitations.countOne
              : t.invitations.countMany.replace(
                  "{count}",
                  invitations.length.toString(),
                )}
          </styled.p>

          <HStack gap="2">
            <Button onClick={handleCreate} loading={isCreating}>
              <AddIcon />
              {t.invitations.create}
            </Button>
            <Button variant="outline" onClick={handleOpenMultiForm}>
              <AddIcon />
              {t.invitations.createMulti}
            </Button>
          </HStack>
        </WStack>

        {isMultiFormOpen && (
          <MultiUseInvitationForm
            onSuccess={handleMultiCreated}
            onCancel={handleCloseMultiForm}
          />
        )}

        {createdID && (
          <LStack gap="2">
            <styled.p color="fg.muted" fontSize="sm">
              {t.invitations.createdHint}
            </styled.p>

            <InvitationLinkField id={createdID} />

            <Button size="xs" variant="subtle" onClick={handleDismissCreated}>
              {t.actions.done}
            </Button>
          </LStack>
        )}

        {invitations.length === 0 ? (
          <styled.p color="fg.subtle" fontStyle="italic">
            {t.invitations.empty}
          </styled.p>
        ) : (
          <LStack>
            {invitations.map((invitation) => (
              <InvitationItem
                key={invitation.id}
                invitation={invitation}
                onDelete={handleDelete}
              />
            ))}
          </LStack>
        )}
      </LStack>
    </CardBox>
  );
}

function InvitationLinkField({ id }: { id: string }) {
  const t = useTranslation();

  return (
    <Clipboard.Root w="full" value={createInvitationLink(id)}>
      <Clipboard.Control gap="0">
        <Clipboard.Input asChild>
          <Input readOnly aria-label={t.invitations.linkLabel} />
        </Clipboard.Input>
        <Clipboard.Trigger asChild>
          <IconButton variant="outline" aria-label={t.invitations.copy}>
            <Clipboard.Indicator copied={<CheckIcon />}>
              <CopyIcon />
            </Clipboard.Indicator>
          </IconButton>
        </Clipboard.Trigger>
      </Clipboard.Control>
    </Clipboard.Root>
  );
}

function InvitationItem({
  invitation,
  onDelete,
}: {
  invitation: Invitation;
  onDelete: (id: string) => Promise<void>;
}) {
  const t = useTranslation();
  const { isConfirming, handleConfirmAction, handleCancelAction } =
    useConfirmation(() => onDelete(invitation.id));

  const isExpired = invitation.expires_at
    ? new Date(invitation.expires_at) < new Date()
    : false;
  const isExhausted =
    invitation.max_uses != null && invitation.uses >= invitation.max_uses;

  return (
    <WStack
      w="full"
      alignItems="center"
      borderBottomWidth="thin"
      borderBottomStyle="solid"
      borderBottomColor="border.subtle"
      pb="2"
    >
      <LStack gap="0" minW="0">
        <styled.p fontSize="sm">
          {formatDate(new Date(invitation.createdAt), "yyyy-MM-dd HH:mm")}
        </styled.p>
        {invitation.message && (
          <styled.p color="fg.muted" fontSize="sm" lineClamp={1}>
            {invitation.message}
          </styled.p>
        )}
        {(invitation.max_uses != null || invitation.expires_at) && (
          <HStack gap="2" fontSize="xs">
            {invitation.max_uses != null && (
              <styled.span color={isExhausted ? "fg.error" : "fg.muted"}>
                {t.invitations.usesCount
                  .replace("{used}", String(invitation.uses))
                  .replace("{max}", String(invitation.max_uses))}
              </styled.span>
            )}
            {invitation.expires_at && (
              <styled.span color={isExpired ? "fg.error" : "fg.muted"}>
                {isExpired
                  ? t.invitations.expired
                  : t.invitations.expiresOn.replace(
                      "{date}",
                      formatDate(
                        new Date(invitation.expires_at),
                        "yyyy-MM-dd HH:mm",
                      ),
                    )}
              </styled.span>
            )}
          </HStack>
        )}
      </LStack>

      <HStack gap="1">
        <Clipboard.Root value={createInvitationLink(invitation.id)}>
          <Clipboard.Trigger asChild>
            <IconButton
              size="xs"
              variant="ghost"
              aria-label={t.invitations.copy}
            >
              <Clipboard.Indicator copied={<CheckIcon />}>
                <CopyIcon />
              </Clipboard.Indicator>
            </IconButton>
          </Clipboard.Trigger>
        </Clipboard.Root>

        {isConfirming ? (
          <HStack gap="1">
            <Button size="xs" variant="subtle" onClick={handleCancelAction}>
              {t.common.cancel}
            </Button>
            <Button size="xs" variant="solid" onClick={handleConfirmAction}>
              {t.actions.deleteConfirm}
            </Button>
          </HStack>
        ) : (
          <IconButton
            size="xs"
            variant="ghost"
            aria-label={t.actions.delete}
            onClick={handleConfirmAction}
          >
            <DeleteIcon />
          </IconButton>
        )}
      </HStack>
    </WStack>
  );
}
