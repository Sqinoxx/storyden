import { formatDate } from "date-fns";
import { de, enUS } from "date-fns/locale";
import { useMemo } from "react";
import { match } from "ts-pattern";

import { RequestError } from "@/api/common";
import { useAccountView } from "@/api/openapi-client/accounts";
import { Account, Permission, ProfileReference } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { MemberIdent } from "@/components/member/MemberBadge/MemberIdent";
import { Badge } from "@/components/ui/badge";
import { AdminIcon } from "@/components/ui/icons/Admin";
import { WarningIcon } from "@/components/ui/icons/Warning";
import * as Tabs from "@/components/ui/tabs";
import { useLanguage } from "@/lib/i18n";
import {
  Box,
  CardBox,
  Flex,
  HStack,
  LStack,
  styled,
} from "@/styled-system/jsx";
import { deriveError } from "@/utils/error";
import { hasPermission } from "@/utils/permissions";

import { AccountPurgeTrigger } from "./AccountPurgeModal/AccountPurgeTrigger";
import { ModeratorNotesPanel } from "./ModeratorNotesPanel";
import { WarningsPanel } from "./WarningsPanel";

type Props = {
  accountId: string;
};

function useProfileAccountManagement({ accountId }: Props) {
  const { t } = useLanguage();
  const { data: account, error } = useAccountView(accountId);
  if (!account) {
    if (
      error !== undefined &&
      error instanceof RequestError &&
      error.status === 403
    ) {
      return {
        ready: false as const,
        error: t.profile.noPermissionAdmin,
      };
    }

    return {
      ready: false as const,
      error: deriveError(error),
    };
  }

  const emailVerifiedStatus =
    account.email_addresses.length === 0
      ? "not_applicable"
      : account.email_addresses.some((e) => e.verified)
        ? "verified"
        : "not_verified";

  return {
    ready: true as const,
    account,
    emailVerifiedStatus,
  };
}

export function ProfileAccountManagement({ accountId }: Props) {
  const { t } = useLanguage();
  const { ready, error, account, emailVerifiedStatus } =
    useProfileAccountManagement({ accountId });

  if (!ready) {
    return (
      <CardBox
        borderColor="border.warning"
        borderWidth="thin"
        borderStyle="dashed"
        borderRadius="sm"
        p="2"
      >
        <HStack
          alignItems="center"
          color="fg.subtle"
          role="alert"
          aria-atomic="true"
        >
          <Box w="5" flexShrink="0">
            <WarningIcon aria-hidden="true" />
          </Box>
          <p id="error__message">{error}</p>
        </HStack>
      </CardBox>
    );
  }

  return (
    <CardBox
      p="0"
      borderColor="border.warning"
      borderWidth="thin"
      borderStyle="dashed"
      borderRadius="sm"
    >
      <Box bgColor="bg.warning" borderTopRadius="sm" pl="3" pr="2" py="1">
        <HStack
          gap="1"
          color="fg.warning"
          fontSize="xs"
          justifyContent="space-between"
        >
          <HStack gap="1">
            <AdminIcon w="4" />
            <p>{t.profile.accountInfo}</p>
          </HStack>
          <AccountPurgeTrigger accountId={account.id} handle={account.handle} />
        </HStack>
      </Box>

      <ProfileAccountManagementTabs
        account={account}
        emailVerifiedStatus={emailVerifiedStatus}
      />
    </CardBox>
  );
}

function ProfileAccountManagementTabs({
  account,
  emailVerifiedStatus,
}: {
  account: Account;
  emailVerifiedStatus: string;
}) {
  const { language, t } = useLanguage();
  const dateLocale = language === "de" ? de : enUS;
  const session = useSession();
  const canManageWarnings = hasPermission(session, Permission.MANAGE_WARNINGS);
  const isSelf = session?.id === account.id;
  const canViewWarnings = canManageWarnings || isSelf;
  const canViewModerationNotes = hasPermission(
    session,
    Permission.VIEW_MODERATION_NOTES,
  );
  const canManageModerationNotes = hasPermission(
    session,
    Permission.MANAGE_MODERATION_NOTES,
  );
  const hasModerationNotesAccess =
    canViewModerationNotes || canManageModerationNotes;

  const emailVerifiedStatusBadge = match(emailVerifiedStatus)
    .with("not_applicable", () => (
      <Badge colorPalette="gray">{t.profile.noEmailsToVerify}</Badge>
    ))
    .with("verified", () => <Badge colorPalette="green">{t.profile.verified}</Badge>)
    .with("not_verified", () => <Badge colorPalette="gray">{t.profile.unverified}</Badge>);

  return (
    <Tabs.Root
      variant="line"
      size="sm"
      pt="2"
      colorPalette="amber"
      defaultValue="overview"
      w="full"
      lazyMount={true}
    >
      <Tabs.List>
        <Tabs.Trigger value="overview">{t.profile.overview}</Tabs.Trigger>
        {canViewWarnings && (
          <Tabs.Trigger value="warnings">{t.profile.warnings}</Tabs.Trigger>
        )}
        {hasModerationNotesAccess && (
          <Tabs.Trigger value="moderator_notes">{t.profile.moderatorNotes}</Tabs.Trigger>
        )}
        <Tabs.Indicator />
      </Tabs.List>

      <Tabs.Content value="overview" px="3" pb="3">
        <Flex
          gap={{ base: "4", md: "6" }}
          direction={{ base: "column", md: "row" }}
          alignItems="start"
        >
          <LStack flex="1" gap="4" flexShrink="1" flexGrow="1" minW="0">
            <LStack gap="1">
              <styled.p fontSize="xs" fontWeight="semibold" color="fg.muted">
                {t.profile.accountStatus}
              </styled.p>
              <Box fontSize="sm">{emailVerifiedStatusBadge.run()}</Box>
            </LStack>

            <LStack gap="1">
              <styled.p fontSize="xs" fontWeight="semibold" color="fg.muted">
                {t.profile.joinedAt}
              </styled.p>
              <styled.p fontSize="sm">
                {formatDate(new Date(account.joined), "PPPppp", { locale: dateLocale })}
              </styled.p>
            </LStack>

            {account.suspended && (
              <LStack gap="1">
                <styled.p fontSize="xs" fontWeight="semibold" color="fg.muted">
                  {t.profile.suspended}
                </styled.p>
                <styled.p fontSize="sm" color="fg.destructive">
                  {formatDate(new Date(account.suspended), "PPPppp", { locale: dateLocale })}
                </styled.p>
              </LStack>
            )}

            {account.invited_by && (
              <LStack gap="1">
                <styled.p fontSize="xs" fontWeight="semibold" color="fg.muted">
                  {t.members.invitedBy}
                </styled.p>
                <MemberIdent
                  size="sm"
                  name="full-vertical"
                  profile={account.invited_by}
                />
              </LStack>
            )}
          </LStack>

          <LStack flex="1" gap="2" flexShrink="1" flexGrow="1" minW="0">
            <styled.p fontSize="xs" fontWeight="semibold" color="fg.muted">
              {t.profile.emailAddresses}
            </styled.p>
            {account.email_addresses.length > 0 ? (
              <LStack gap="2" minW="0">
                {account.email_addresses.map((email) => (
                  <HStack
                    key={email.id}
                    gap="2"
                    fontSize="sm"
                    flexWrap="wrap"
                    minW="0"
                    width="full"
                  >
                    <styled.code
                      fontFamily="mono"
                      w="full"
                      minW="0"
                      textOverflow="ellipsis"
                      overflow="hidden"
                    >
                      {email.email_address}
                    </styled.code>
                    {email.verified ? (
                      <Badge colorPalette="green" size="sm">
                        {t.profile.verified}
                      </Badge>
                    ) : (
                      <Badge colorPalette="gray" size="sm">
                        {t.profile.unverified}
                      </Badge>
                    )}
                  </HStack>
                ))}
              </LStack>
            ) : (
              <styled.p fontSize="sm" color="fg.subtle">
                {t.settings.email.noEmails}
              </styled.p>
            )}
          </LStack>
        </Flex>
      </Tabs.Content>

      {canViewWarnings && (
        <Tabs.Content value="warnings" px="3" pb="3">
          <WarningsPanel
            accountId={account.id}
            profile={account}
            canManageWarnings={canManageWarnings}
          />
        </Tabs.Content>
      )}

      {hasModerationNotesAccess && (
        <Tabs.Content value="moderator_notes" px="3" pb="3">
          <ModeratorNotesPanel
            accountId={account.id}
            canManageModerationNotes={canManageModerationNotes}
            canViewModerationNotes={canViewModerationNotes}
          />
        </Tabs.Content>
      )}
    </Tabs.Root>
  );
}
