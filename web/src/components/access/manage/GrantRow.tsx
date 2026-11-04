"use client";

import Link from "next/link";

import { AccessGrant } from "@/api/openapi-schema";
import { Badge } from "@/components/ui/badge";
import { useLanguage } from "@/lib/i18n";
import { HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";

export function formatAccessDate(iso: string | undefined, language: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString(language, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

type Props = {
  grant: AccessGrant;
  showStatus?: boolean;
  actions?: React.ReactNode;
};

export function GrantRow({ grant, showStatus, actions }: Props) {
  const { t, language } = useLanguage();
  const m = t.access.manage;

  const when =
    grant.status === "requested"
      ? m.requestedOn.replace(
          "{date}",
          formatAccessDate(grant.requested_at, language),
        )
      : grant.expires_at
        ? m.untilDate.replace(
            "{date}",
            formatAccessDate(grant.expires_at, language),
          )
        : grant.status === "active"
          ? m.noExpiry
          : "";

  return (
    <WStack
      className={CardBox()}
      p="3"
      gap="3"
      alignItems="center"
      flexWrap="wrap"
      aria-label={`${grant.account.name} ${grant.pass.name}`}
      data-testid="access-grant-row"
    >
      <LStack gap="1" minW="0" flex="1">
        <HStack gap="2" flexWrap="wrap">
          <Link href={`/m/${grant.account.handle}`}>
            <styled.span fontWeight="semibold">
              {grant.account.name}
            </styled.span>{" "}
            <styled.span color="fg.muted">@{grant.account.handle}</styled.span>
          </Link>
          {showStatus && (
            <Badge size="sm" variant="subtle">
              {t.access.status[grant.status]}
            </Badge>
          )}
        </HStack>
        <HStack gap="2" flexWrap="wrap" fontSize="sm" color="fg.muted">
          <styled.span>{grant.pass.name}</styled.span>
          <styled.code data-testid="access-grant-reference">
            {grant.reference_code}
          </styled.code>
          {when && <styled.span>{when}</styled.span>}
          {grant.decided_by && grant.status !== "requested" && (
            <styled.span>
              {m.decidedBy.replace("{name}", grant.decided_by.name)}
            </styled.span>
          )}
        </HStack>
        {grant.note && (
          <styled.p fontSize="sm" color="fg.subtle">
            {grant.note}
          </styled.p>
        )}
      </LStack>
      {actions && (
        <HStack gap="2" flexShrink="0">
          {actions}
        </HStack>
      )}
    </WStack>
  );
}
