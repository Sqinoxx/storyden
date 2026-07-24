"use client";

import { useSession } from "@/auth";
import {
  LoginAnchor,
  RegisterAnchor,
} from "@/components/site/Navigation/Anchors/Login";

import { Account } from "@/api/openapi-schema";
import { NotificationsMenu } from "@/components/notifications/NotificationsMenu";
import { HStack } from "@/styled-system/jsx";

import { AccountMenu } from "./AccountMenu/AccountMenu";
import { ComposeAnchor } from "./Anchors/Compose";
import { useTranslation } from "@/lib/i18n";

type Props = {
  session: Account | undefined;
  canRegister?: boolean;
};

export function MemberActions({ session, canRegister }: Props) {
  const account = useSession(session);
  const t = useTranslation();
  return (
    <HStack w="full" gap="2" alignItems="center" justify="end" pr="1">
      {account ? (
        <>
          <ComposeAnchor>{t.nav.post}</ComposeAnchor>
          <NotificationsMenu status="unread" />
          <AccountMenu account={account} />
        </>
      ) : (
        <>
          {canRegister && <RegisterAnchor w="full" />}
          <LoginAnchor flexShrink={0} />
        </>
      )}
    </HStack>
  );
}
