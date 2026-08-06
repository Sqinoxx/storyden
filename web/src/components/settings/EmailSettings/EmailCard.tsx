"use client";

import Link from "next/link";
import { useSWRConfig } from "swr";

import { handle } from "@/api/client";
import {
  accountEmailRemove,
  getAccountGetKey,
} from "@/api/openapi-client/accounts";
import { AccountEmailAddress } from "@/api/openapi-schema";
import { CancelAction } from "@/components/site/Action/Cancel";
import { useConfirmation } from "@/components/site/useConfirmation";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { CardBox, HStack, WStack } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

type Props = {
  email: AccountEmailAddress;
};
export function EmailCard({ email }: Props) {
  const { t } = useLanguage();
  const { mutate } = useSWRConfig();

  async function handleRemove() {
    await handle(
      async () => {
        await accountEmailRemove(email.id);
      },
      {
        async cleanup() {
          await mutate(getAccountGetKey());
        },
        promiseToast: {
          loading: t.settings?.email?.deletingEmail || "Deleting email address...",
          success: t.settings?.email?.emailDeleted || "Email deleted",
        },
      },
    );
  }

  const { isConfirming, handleCancelAction, handleConfirmAction } =
    useConfirmation(handleRemove);

  return (
    <CardBox key={email.email_address} className={lstack()} gap="4">
      <WStack alignItems="center">
        <HStack>
          <Heading size="sm">{email.email_address}</Heading>
          {email.verified ? (
            <Badge
              borderColor="border.success"
              backgroundColor="bg.success"
              color="fg.success"
            >
              {t.settings?.email?.verified || "Verified"}
            </Badge>
          ) : (
            <Link href="/auth/verify/email?returnURL=/settings">
              <Badge borderColor="border.error" backgroundColor="bg.error" color="fg.error">
                {t.settings?.email?.verifyThisEmail || "Verify this email"}
              </Badge>
            </Link>
          )}
        </HStack>

        <HStack gap="0">
          <Button
            style={{
              borderBottomRightRadius: isConfirming ? "0" : undefined,
              borderTopRightRadius: isConfirming ? "0" : undefined,
            }}
            size="xs"
            variant="subtle"
            onClick={handleConfirmAction}
          >
            {isConfirming ? (
              t.actions?.deleteConfirm || "Are you sure?"
            ) : (
              <>
                <DeleteIcon /> {t.settings?.email?.deleteEmail || "delete email"}
              </>
            )}
          </Button>

          {isConfirming && (
            <CancelAction
              variant="subtle"
              borderLeftRadius="none"
              onClick={handleCancelAction}
            />
          )}
        </HStack>
      </WStack>
    </CardBox>
  );
}
