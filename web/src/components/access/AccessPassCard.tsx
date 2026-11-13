"use client";

import {
  AlertCircle,
  Check,
  Clock,
  KeyRound,
  Lock,
  LockOpen,
} from "lucide-react";
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
import { LinkButton } from "@/components/ui/link-button";
import { membershipAction } from "@/lib/access/access";
import { useAccessPasses } from "@/lib/access/useAccess";
import { useLanguage } from "@/lib/i18n";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
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

function descriptionLines(description: string | undefined) {
  return (description ?? "")
    .split("\n")
    .map((l) => l.replace(/^[-*•]\s*/, "").trim())
    .filter(Boolean);
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
  const lines = descriptionLines(pass.description);

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

  const warning =
    status === "expired"
      ? t.access.expired.replace(
          "{date}",
          formatDate(membership?.expires_at, language),
        )
      : status === "rejected"
        ? t.access.rejected
        : status === "revoked"
          ? t.access.revoked
          : undefined;

  return (
    <styled.section
      className={CardBox()}
      p="0"
      gap="0"
      w="full"
      overflow="hidden"
      borderWidth="thin"
      borderColor={isActive ? "border.success" : "border.subtle"}
      borderRadius="xl"
      aria-label={pass.name}
      data-testid="access-pass-card"
    >
      <HStack
        gap="3"
        alignItems="center"
        w="full"
        p="4"
        bg={isActive ? "bg.success" : "accent.1"}
        borderBottomWidth="thin"
        borderBottomColor="border.subtle"
      >
        <IconTile active={isActive} />
        <LStack gap="0.5" minW="0" flex="1">
          <styled.span
            fontSize="xs"
            fontWeight="semibold"
            textTransform="uppercase"
            letterSpacing="wide"
            color={isActive ? "fg.success" : "fg.accent"}
          >
            {status ? t.access.status[status] : t.access.locked}
          </styled.span>
          <Heading size="md" fontWeight="bold" overflowWrap="anywhere">
            {pass.name}
          </Heading>
        </LStack>
        {pass.price_label && (
          <styled.span
            flexShrink="0"
            px="3"
            py="1"
            borderRadius="full"
            bg="bg.default"
            borderWidth="thin"
            borderColor="border.subtle"
            fontSize="lg"
            fontWeight="bold"
            whiteSpace="nowrap"
          >
            {pass.price_label}
          </styled.span>
        )}
      </HStack>

      <LStack gap="4" w="full" p="4">
        {lines.length > 1 ? (
          <styled.ul display="flex" flexDir="column" gap="1.5" w="full">
            {lines.map((line, i) => (
              <styled.li
                key={i}
                display="flex"
                gap="2"
                alignItems="start"
                listStyle="none"
                overflowWrap="anywhere"
              >
                <styled.span flexShrink="0" pt="0.5" color="fg.accent">
                  <Check size={16} />
                </styled.span>
                {line}
              </styled.li>
            ))}
          </styled.ul>
        ) : (
          pass.description && (
            <styled.p
              color="fg.muted"
              whiteSpace="pre-wrap"
              overflowWrap="anywhere"
            >
              {pass.description}
            </styled.p>
          )
        )}

        {categoryNames && categoryNames.length > 0 && (
          <HStack gap="1.5" flexWrap="wrap" w="full">
            <styled.span fontSize="sm" color="fg.muted">
              {t.access.includes}:
            </styled.span>
            {categoryNames.map((name) => (
              <Badge key={name} size="sm" variant="subtle">
                {name}
              </Badge>
            ))}
          </HStack>
        )}

        {isActive ? (
          <HStack gap="2" fontSize="sm" color="fg.success">
            <LockOpen size={16} />
            <styled.span>
              {membership?.expires_at
                ? t.access.activeUntil.replace(
                    "{date}",
                    formatDate(membership.expires_at, language),
                  )
                : t.access.activeForever}
            </styled.span>
          </HStack>
        ) : (
          <>
            {warning && (
              <HStack
                gap="2"
                alignItems="start"
                w="full"
                p="3"
                borderRadius="lg"
                bg="bg.warning"
                color="fg.warning"
                fontSize="sm"
              >
                <styled.span flexShrink="0" pt="0.5">
                  <AlertCircle size={16} />
                </styled.span>
                <LStack gap="1" minW="0">
                  <styled.p>{warning}</styled.p>
                  {(status === "rejected" || status === "revoked") &&
                    membership?.reason && (
                      <styled.p
                        overflowWrap="anywhere"
                        data-testid="access-reason"
                      >
                        {t.access.reason.replace("{reason}", membership.reason)}
                      </styled.p>
                    )}
                </LStack>
              </HStack>
            )}

            <styled.ol
              display="flex"
              flexDir="column"
              w="full"
              p="3"
              borderRadius="lg"
              bg="bg.subtle"
            >
              <Step
                index={1}
                state={isRequested ? "done" : "current"}
                label={t.access.stepRequest}
              />
              <Step
                index={2}
                state={isRequested ? "current" : "upcoming"}
                label={
                  pass.price_label
                    ? t.access.stepPayPrice.replace("{price}", pass.price_label)
                    : t.access.stepPay
                }
              >
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
                {session && (
                  <styled.p
                    fontSize="sm"
                    color="fg.muted"
                    data-testid="access-tell-handle"
                  >
                    {t.access.tellHandle.split("{handle}")[0]}
                    <styled.code
                      fontWeight="bold"
                      px="1.5"
                      py="0.5"
                      borderRadius="md"
                      bg="bg.default"
                      color="fg.default"
                    >
                      @{session.handle}
                    </styled.code>
                    {t.access.tellHandle.split("{handle}")[1]}
                  </styled.p>
                )}
              </Step>
              <Step index={3} state="upcoming" label={t.access.stepWait} last />
            </styled.ol>

            {isRequested && membership?.requested_at && (
              <HStack gap="2" alignItems="start" fontSize="sm" color="fg.muted">
                <styled.span flexShrink="0" pt="0.5">
                  <Clock size={14} />
                </styled.span>
                <styled.span>
                  {t.access.requested.replace(
                    "{date}",
                    formatDate(membership.requested_at, language),
                  )}
                </styled.span>
              </HStack>
            )}

            {action === "login" && (
              <LinkButton href="/login" w={{ base: "full", sm: "auto" }}>
                <KeyRound />
                {t.access.loginToRequest}
              </LinkButton>
            )}
            {action === "request" && (
              <Button
                w={{ base: "full", sm: "auto" }}
                loading={busy}
                onClick={onRequest}
              >
                <KeyRound />
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
          </>
        )}
      </LStack>
    </styled.section>
  );
}

function IconTile({ active, small }: { active: boolean; small?: boolean }) {
  const Icon = active ? LockOpen : Lock;
  return (
    <styled.span
      display="flex"
      alignItems="center"
      justifyContent="center"
      flexShrink="0"
      w={small ? "8" : "10"}
      h={small ? "8" : "10"}
      borderRadius="lg"
      bg={active ? "bg.default" : "accent.3"}
      color={active ? "fg.success" : "fg.accent"}
    >
      <Icon size={small ? 16 : 20} />
    </styled.span>
  );
}

type StepState = "done" | "current" | "upcoming";

function Step({
  index,
  state,
  label,
  last,
  children,
}: {
  index: number;
  state: StepState;
  label: string;
  last?: boolean;
  children?: React.ReactNode;
}) {
  return (
    <styled.li display="flex" gap="3" alignItems="stretch" listStyle="none">
      <styled.span
        display="flex"
        flexDir="column"
        alignItems="center"
        flexShrink="0"
      >
        <styled.span
          display="flex"
          alignItems="center"
          justifyContent="center"
          w="6"
          h="6"
          borderRadius="full"
          borderWidth="medium"
          fontSize="xs"
          fontWeight="bold"
          borderColor={
            state === "upcoming" ? "border.default" : "border.accent"
          }
          bg={state === "done" ? "bg.accent" : "bg.default"}
          color={
            state === "done"
              ? "fg.default"
              : state === "current"
                ? "fg.accent"
                : "fg.muted"
          }
        >
          {state === "done" ? <Check size={14} /> : index}
        </styled.span>
        {!last && (
          <Box
            flex="1"
            minH="3"
            borderLeftWidth="medium"
            borderLeftColor={
              state === "done" ? "border.accent" : "border.default"
            }
          />
        )}
      </styled.span>
      <LStack gap="1" minW="0" flex="1" pb={last ? "0" : "3"} pt="0.5">
        <styled.span
          fontWeight={state === "current" ? "semibold" : "medium"}
          color={state === "upcoming" ? "fg.muted" : "fg.default"}
        >
          {label}
        </styled.span>
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
        gap="3"
        alignItems="center"
        w="full"
        p="4"
        borderRadius="xl"
        borderWidth="thin"
        borderColor="border.subtle"
        bg="accent.1"
        color="fg.muted"
        data-testid="access-locked-notice"
      >
        <IconTile active={false} small />
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
