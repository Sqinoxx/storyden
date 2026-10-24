import { useState } from "react";
import { useSWRConfig } from "swr";

import { handle } from "@/api/client";
import {
  getAccountViewKey,
  useAccountManageUpdateEmail,
} from "@/api/openapi-client/accounts";
import { AccountEmailAddress } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  accountId: string;
  email: AccountEmailAddress;
  canEdit: boolean;
};

export function AdminEmailEditor({ accountId, email, canEdit }: Props) {
  const t = useTranslation();
  const { mutate } = useSWRConfig();
  const [editing, setEditing] = useState(false);
  const [address, setAddress] = useState(email.email_address);
  const [verified, setVerified] = useState(email.verified);
  const { trigger, isMutating } = useAccountManageUpdateEmail(
    accountId,
    email.id,
  );

  function startEditing() {
    setAddress(email.email_address);
    setVerified(email.verified);
    setEditing(true);
  }

  async function save() {
    await handle(
      async () => {
        await trigger({ email_address: address.trim(), verified });
        await mutate(getAccountViewKey(accountId));
        setEditing(false);
      },
      {
        promiseToast: {
          loading: t.profile.savingEmail,
          success: t.profile.emailUpdated,
        },
      },
    );
  }

  if (!editing) {
    return (
      <HStack
        gap="2"
        fontSize="sm"
        flexWrap="wrap"
        minW="0"
        width="full"
      >
        <styled.code
          fontFamily="mono"
          flex="1"
          minW="0"
          textOverflow="ellipsis"
          overflow="hidden"
        >
          {email.email_address}
        </styled.code>
        <Badge colorPalette={email.verified ? "green" : "gray"} size="sm">
          {email.verified ? t.profile.verified : t.profile.unverified}
        </Badge>
        {canEdit && (
          <Button size="xs" variant="ghost" onClick={startEditing}>
            {t.profile.editEmail}
          </Button>
        )}
      </HStack>
    );
  }

  return (
    <LStack gap="2" minW="0" width="full">
      <Input
        type="email"
        size="sm"
        value={address}
        onChange={(e) => setAddress(e.target.value)}
        aria-label={t.profile.editEmail}
      />
      <styled.label display="flex" alignItems="center" gap="2" fontSize="sm">
        <input
          type="checkbox"
          checked={verified}
          onChange={(e) => setVerified(e.target.checked)}
        />
        {t.profile.markEmailVerified}
      </styled.label>
      <HStack gap="2">
        <Button
          size="xs"
          variant="subtle"
          onClick={save}
          loading={isMutating}
          disabled={!address.trim()}
        >
          {t.profile.saveEmail}
        </Button>
        <Button
          size="xs"
          variant="ghost"
          onClick={() => setEditing(false)}
          disabled={isMutating}
        >
          {t.profile.cancelEmailEdit}
        </Button>
      </HStack>
    </LStack>
  );
}
