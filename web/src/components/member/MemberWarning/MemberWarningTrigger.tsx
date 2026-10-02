import React, { ReactNode, useState } from "react";
import { toast } from "sonner";
import { useSWRConfig } from "swr";

import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { useDisclosure } from "@/utils/useDisclosure";

import { handle } from "@/api/client";
import {
  getAccountWarningListKey,
  useAccountWarningCreate,
} from "@/api/openapi-client/accounts";
import { ProfileReference } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { HStack, VStack, styled } from "@/styled-system/jsx";
import { useTranslation } from "@/lib/i18n";

type MemberWarningTriggerProps = {
  children?: ReactNode;
  profile: ProfileReference;
};

export function MemberWarningTrigger({
  children,
  profile,
}: MemberWarningTriggerProps) {
  const t = useTranslation();
  const { mutate } = useSWRConfig();
  const { onOpen, onClose, isOpen } = useDisclosure();
  const { trigger: createWarning, isMutating: loading } =
    useAccountWarningCreate(profile.id);
  const [reason, setReason] = useState("");

  async function issueWarning() {
    if (!reason.trim()) {
      toast.error(t.moderation.warningReasonRequired);
      return;
    }

    await handle(async () => {
      await createWarning({
        reason: reason.trim(),
      });
      await mutate(getAccountWarningListKey(profile.id));
      toast.success(t.moderation.warningIssued.replace("{name}", profile.name));
      setReason("");
      onClose();
    });
  }

  const triggerNode = React.isValidElement<{
    onClick?: React.MouseEventHandler;
  }>(children) ? (
    React.cloneElement(children, { onClick: onOpen })
  ) : (
    <Button colorPalette="orange" onClick={onOpen}>
      {t.moderation.warn}
    </Button>
  );

  return (
    <>
      {triggerNode}
      <ModalDrawer
        isOpen={isOpen}
        onClose={onClose}
        title={t.moderation.issueWarningTo.replace("{name}", profile.name)}
      >
        <VStack alignItems="start" gap="3">
          <styled.p fontSize="sm" color="fg.subtle">
            {t.moderation.warningsRecorded}
          </styled.p>
          <styled.textarea
            rows={5}
            value={reason}
            onChange={(e) => setReason(e.currentTarget.value)}
            placeholder={t.moderation.warningReasonPlaceholder}
            width="full"
            borderWidth="thin"
            borderRadius="sm"
            borderColor="border.default"
            padding="2"
          />

          <HStack w="full">
            <Button
              type="button"
              flexGrow="1"
              onClick={onClose}
              disabled={loading}
            >
              {t.common.cancel}
            </Button>
            <Button
              type="button"
              flexGrow="1"
              colorPalette="orange"
              onClick={issueWarning}
              loading={loading}
            >
              {t.profile.issueWarning}
            </Button>
          </HStack>
        </VStack>
      </ModalDrawer>
    </>
  );
}
