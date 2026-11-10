"use client";

import { Check, Copy, Lock, LockOpen } from "lucide-react";
import Link from "next/link";
import { useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import {
  accessPassRequest,
  accessPassRequestWithdraw,
} from "@/api/openapi-client/access";
import { AccessPass } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { IconButton } from "@/components/ui/icon-button";
import { LinkButton } from "@/components/ui/link-button";
import { membershipAction } from "@/lib/access/access";
import { useAccessPasses } from "@/lib/access/useAccess";
import { useLanguage } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";
import { CardBox } from "@/styled-system/patterns";

type Props = {
  pass: AccessPass;
  categoryNames?: string[];
};

function formatDate(iso: string | undefined, language: string) {
  if (!iso) return "";
  return new Date(iso).toLocaleDateString(language, {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
  });
}

export function AccessPassCard({ pass, categoryNames }: Props) {
  const session = useSession();
  const { t, language } = useLanguage();
  const { mutate } = useAccessPasses();
  const [busy, setBusy] = useState(false);

  const membership = pass.membership;
  const status = membership?.status;
  const action = membershipAction(pass, Boolean(session));
  const isActive = status === "active";
  const isRequested = status === "requested";

  async function onRequest() {
    setBusy(true);
    await handle(
      async () => {
        await accessPassRequest(pass.id);
        await mutate();
        toast.success(t.access.requestSent);
      },
      { cleanup: async () => setBusy(false) },
    );
  }

  async function onWithdraw() {
    setBusy(true);
    await handle(
      async () => {
        await accessPassRequestWithdraw(pass.id);
        await mutate();
        toast.success(t.access.requestWithdrawn);
      },
      { cleanup: async () => setBusy(false) },
    );
  }

  async function onCopy() {
    if (!membership) return;
    try {
      await navigator.clipboard.writeText(membership.reference_code);
      toast.success(t.access.copied);
    } catch {
      toast.error(t.access.copyFailed);
    }
  }

  return (
    <styled.section
      className={CardBox()}
      p="4"
      gap="3"
      w="full"
      aria-label={pass.name}
      data-testid="access-pass-card"
    >
      <HStack
        justify="space-between"
        alignItems="start"
        gap="2"
        w="full"
        flexWrap="wrap"
      >
        <HStack gap="2" minW="0" alignItems="start">
          <styled.span flexShrink="0" pt="0.5">
            {isActive ? <LockOpen size={18} /> : <Lock size={18} />}
          </styled.span>
          <Heading size="md" fontWeight="bold" overflowWrap="anywhere">
            {pass.name}
          </Heading>
        </HStack>
        <HStack gap="2" flexShrink="0" flexWrap="wrap">
          {status && (
            <Badge size="sm" variant="subtle">
              {t.access.status[status]}
            </Badge>
          )}
          {pass.price_label && (
            <Badge size="sm" variant="outline">
              {pass.price_label}
            </Badge>
          )}
        </HStack>
      </HStack>

      {pass.description && (
        <styled.p
          color="fg.muted"
          whiteSpace="pre-wrap"
          overflowWrap="anywhere"
        >
          {pass.description}
        </styled.p>
      )}

      {categoryNames && categoryNames.length > 0 && (
        <styled.p fontSize="sm" color="fg.muted">
          {t.access.includes}: {categoryNames.join(", ")}
        </styled.p>
      )}

      {isActive ? (
        <styled.p fontSize="sm" color="fg.success">
          {membership?.expires_at
            ? t.access.activeUntil.replace(
                "{date}",
                formatDate(membership.expires_at, language),
              )
            : t.access.activeForever}
        </styled.p>
      ) : (
        <>
          {status === "expired" && (
            <styled.p fontSize="sm" color="fg.warning">
              {t.access.expired.replace(
                "{date}",
                formatDate(membership?.expires_at, language),
              )}
            </styled.p>
          )}
          {status === "rejected" && (
            <styled.p fontSize="sm" color="fg.warning">
              {t.access.rejected}
            </styled.p>
          )}
          {status === "revoked" && (
            <styled.p fontSize="sm" color="fg.warning">
              {t.access.revoked}
            </styled.p>
          )}

          <styled.ol display="flex" flexDir="column" gap="2" w="full">
            <Step index={1} done={isRequested} label={t.access.stepRequest} />
            <Step index={2} done={false} label={t.access.stepPay}>
              {pass.payment_info && (
                <styled.p
                  fontSize="sm"
                  color="fg.muted"
                  whiteSpace="pre-wrap"
                  overflowWrap="anywhere"
                >
                  {pass.payment_info}
                </styled.p>
              )}
              {isRequested && membership && (
                <HStack gap="2" flexWrap="wrap">
                  <styled.span fontSize="sm" color="fg.muted">
                    {t.access.referenceCode}:
                  </styled.span>
                  <styled.code
                    fontWeight="bold"
                    data-testid="access-reference-code"
                  >
                    {membership.reference_code}
                  </styled.code>
                  <IconButton
                    size="xs"
                    variant="ghost"
                    onClick={onCopy}
                    aria-label={t.access.copy}
                    title={t.access.copy}
                  >
                    <Copy />
                  </IconButton>
                </HStack>
              )}
            </Step>
            <Step index={3} done={false} label={t.access.stepWait} />
          </styled.ol>

          {isRequested && membership?.requested_at && (
            <styled.p fontSize="sm" color="fg.muted">
              {t.access.requested.replace(
                "{date}",
                formatDate(membership.requested_at, language),
              )}
            </styled.p>
          )}

          <HStack gap="2" flexWrap="wrap" w="full">
            {action === "login" && (
              <LinkButton href="/login" size="sm">
                {t.access.loginToRequest}
              </LinkButton>
            )}
            {action === "request" && (
              <Button size="sm" loading={busy} onClick={onRequest}>
                {t.access.request}
              </Button>
            )}
            {action === "withdraw" && (
              <Button
                size="sm"
                variant="ghost"
                loading={busy}
                onClick={onWithdraw}
              >
                {t.access.withdraw}
              </Button>
            )}
            {action === "none" && !pass.requestable && (
              <styled.p fontSize="sm" color="fg.muted">
                {t.access.contactDirectly}
              </styled.p>
            )}
          </HStack>
        </>
      )}
    </styled.section>
  );
}

function Step({
  index,
  done,
  label,
  children,
}: {
  index: number;
  done: boolean;
  label: string;
  children?: React.ReactNode;
}) {
  return (
    <styled.li display="flex" gap="2" alignItems="start" listStyle="none">
      <styled.span
        display="flex"
        alignItems="center"
        justifyContent="center"
        flexShrink="0"
        w="6"
        h="6"
        borderRadius="full"
        borderWidth="thin"
        borderColor={done ? "border.accent" : "border.default"}
        bg={done ? "bg.accent" : undefined}
        fontSize="xs"
        fontWeight="bold"
      >
        {done ? <Check size={14} /> : index}
      </styled.span>
      <LStack gap="1" minW="0" flex="1">
        <styled.span fontWeight="medium">{label}</styled.span>
        {children}
      </LStack>
    </styled.li>
  );
}

type LockedProps = {
  passIDs: string[];
  fallbackNames?: string[];
};

// Shown in place of withheld content. Falls back to the pass names carried on
// the content itself while the full pass list loads.
export function LockedContentNotice({ passIDs, fallbackNames }: LockedProps) {
  const { t } = useLanguage();
  const { passes } = useAccessPasses();
  const matching = passes.filter((p) => passIDs.includes(p.id));

  if (matching.length === 0) {
    return (
      <HStack
        gap="2"
        alignItems="start"
        color="fg.muted"
        data-testid="access-locked-notice"
      >
        <styled.span flexShrink="0" pt="1">
          <Lock size={16} />
        </styled.span>
        <styled.p>
          {t.access.lockedContent.replace(
            "{pass}",
            (fallbackNames ?? []).join(", "),
          )}
        </styled.p>
      </HStack>
    );
  }

  return (
    <LStack gap="3" w="full" data-testid="access-locked-notice">
      {matching.map((p) => (
        <AccessPassCard key={p.id} pass={p} />
      ))}
    </LStack>
  );
}

export function AccessPassesLink() {
  const { t } = useLanguage();
  return (
    <Link href="/access">
      <styled.span textDecoration="underline">{t.access.title}</styled.span>
    </Link>
  );
}
