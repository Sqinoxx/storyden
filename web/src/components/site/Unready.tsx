"use client";

import { PropsWithChildren } from "react";
import { Lock, ShieldAlert } from "lucide-react";

import { useSession } from "@/auth";
import { useTranslation } from "@/lib/i18n";
import { usePublicRegistration } from "@/lib/settings/registration";
import type { Settings } from "@/lib/settings/settings";
import { ImpressumLink } from "@/components/site/ImpressumModal/ImpressumLink";
import { CookieLink } from "@/components/site/CookieNotice/CookieLink";
import {
  CardBox,
  Center,
  HStack,
  LStack,
  styled,
  VStack,
} from "@/styled-system/jsx";
import { deriveError } from "@/utils/error";

import { Spinner } from "../ui/Spinner";
import { WarningIcon } from "../ui/icons/Warning";
import { LinkButton } from "../ui/link-button";

type Props = {
  error?: unknown;
};

export function Unready({ error }: Props) {
  const t = useTranslation();
  const session = useSession();
  const canRegister = usePublicRegistration();

  if (!error) {
    return (
      <Center
        w="full"
        minH="60"
        role="status"
        aria-busy="true"
        aria-live="polite"
        aria-label="Loading"
      >
        <div aria-hidden="true">
          <Spinner />
        </div>
      </Center>
    );
  }

  const message = deriveError(error);
  const lowerMsg = message.toLowerCase();

  const isAuthError =
    !session ||
    lowerMsg.includes("access") ||
    lowerMsg.includes("unauthoris") ||
    lowerMsg.includes("unauthoriz") ||
    lowerMsg.includes("forbidden") ||
    lowerMsg.includes("permission") ||
    lowerMsg.includes("log in") ||
    lowerMsg.includes("anmelden");

  if (isAuthError) {
    return (
      <Center w="full" py={{ base: "10", md: "16" }} px="4">
        <VStack gap="6" alignItems="center">
          <CardBox
            maxW="md"
            p={{ base: "6", md: "8" }}
            textAlign="center"
            borderRadius="2xl"
            boxShadow="lg"
            borderWidth="thin"
            borderStyle="solid"
            borderColor="border.subtle"
            bg="bg.default"
          >
            <VStack gap="6" alignItems="center">
              <Center
                w="16"
                h="16"
                borderRadius="full"
                bg="accent.subtle"
                color="fg.accent"
                shadow="sm"
              >
                <Lock width={30} height={30} />
              </Center>

              <VStack gap="2" textAlign="center">
                <styled.h2
                  fontSize="xl"
                  fontWeight="bold"
                  color="fg.default"
                  letterSpacing="tight"
                >
                  {t.auth?.loginRequired ?? "Anmeldung erforderlich"}
                </styled.h2>
                <styled.p
                  fontSize="sm"
                  color="fg.muted"
                  maxW="sm"
                  lineHeight="relaxed"
                >
                  {t.auth?.loginRequiredDescription ??
                    "Diese Plattform ist privat. Bitte melde dich an, um auf Themen und Diskussionen zuzugreifen."}
                </styled.p>
              </VStack>

              <HStack w="full" gap="3" pt="2" justifyContent="center">
                <LinkButton href="/login" variant="subtle" size="sm" px="6" minW="32">
                  {t.nav.login}
                </LinkButton>
                {canRegister && (
                  <LinkButton href="/register" variant="outline" size="sm" px="6" minW="32">
                    {t.auth?.register ?? "Registrieren"}
                  </LinkButton>
                )}
              </HStack>
            </VStack>
          </CardBox>

          <HStack color="fg.subtle" fontSize="xs" flexShrink="0" py="1" gap="2">
            <ImpressumLink />
            <styled.span color="fg.subtle">•</styled.span>
            <CookieLink />
            <styled.span color="fg.subtle">•</styled.span>
            <p>powered by storyden</p>
          </HStack>
        </VStack>
      </Center>
    );
  }

  return (
    <Center w="full" py={{ base: "10", md: "16" }} px="4">
      <CardBox
        maxW="md"
        p={{ base: "6", md: "8" }}
        textAlign="center"
        borderRadius="2xl"
        boxShadow="md"
        borderWidth="thin"
        borderStyle="solid"
        borderColor="border.subtle"
        bg="bg.default"
      >
        <VStack gap="5" alignItems="center">
          <Center
            w="14"
            h="14"
            borderRadius="full"
            bg="bg.subtle"
            color="fg.error"
            shadow="xs"
          >
            <ShieldAlert width={26} height={26} />
          </Center>

          <VStack gap="2" textAlign="center">
            <styled.h2 fontSize="lg" fontWeight="bold" color="fg.default">
              {t.common.error}
            </styled.h2>
            <styled.p fontSize="sm" color="fg.muted" maxW="sm" id="error__message">
              {message}
            </styled.p>
          </VStack>
        </VStack>
      </CardBox>
    </Center>
  );
}

export function UnreadyBanner({ error, children }: PropsWithChildren<Props>) {
  const t = useTranslation();

  if (!error) {
    return (
      <Center
        w="full"
        height="96"
        role="status"
        aria-busy="true"
        aria-live="polite"
        aria-label="Loading"
      >
        <Spinner aria-hidden="true" />
      </Center>
    );
  }

  const message = deriveError(error);

  return (
    <Center
      width="full"
      justifyContent="center"
      role="alert"
      aria-atomic="true"
      py="8"
    >
      <CardBox maxW="md" p="6" borderRadius="xl">
        <LStack gap="4">
          <HStack id="error__heading" gap="2" alignItems="center">
            <WarningIcon aria-hidden />
            <styled.h1 fontSize="md" fontWeight="bold" my="0">
              {t.common.error}
            </styled.h1>
          </HStack>

          <styled.p id="error__message">
            <span>{message}</span>
          </styled.p>

          {children && <LStack>{children}</LStack>}
        </LStack>
      </CardBox>
    </Center>
  );
}

export function UnauthenticatedBanner({
  initialSettings,
}: {
  initialSettings?: Settings;
}) {
  const canRegister = usePublicRegistration(initialSettings);
  const t = useTranslation();

  return (
    <Center w="full" py={{ base: "10", md: "16" }} px="4">
      <CardBox
        maxW="md"
        p={{ base: "6", md: "8" }}
        textAlign="center"
        borderRadius="2xl"
        boxShadow="lg"
        borderWidth="thin"
        borderStyle="solid"
        borderColor="border.subtle"
        bg="bg.default"
      >
        <VStack gap="6" alignItems="center">
          <Center
            w="16"
            h="16"
            borderRadius="full"
            bg="accent.subtle"
            color="fg.accent"
            shadow="sm"
          >
            <Lock width={30} height={30} />
          </Center>

          <VStack gap="2" textAlign="center">
            <styled.h2
              fontSize="xl"
              fontWeight="bold"
              color="fg.default"
              letterSpacing="tight"
            >
              {t.auth?.loginRequired ?? "Anmeldung erforderlich"}
            </styled.h2>
            <styled.p
              fontSize="sm"
              color="fg.muted"
              maxW="sm"
              lineHeight="relaxed"
            >
              {t.auth?.loginRequiredDescription ??
                "Diese Plattform ist privat. Bitte melde dich an, um auf Themen und Diskussionen zuzugreifen."}
            </styled.p>
          </VStack>

          <HStack w="full" gap="3" pt="2" justifyContent="center">
            <LinkButton href="/login" variant="subtle" size="sm" px="6" minW="32">
              {t.nav.login}
            </LinkButton>
            {canRegister && (
              <LinkButton href="/register" variant="outline" size="sm" px="6" minW="32">
                {t.auth?.register ?? "Registrieren"}
              </LinkButton>
            )}
          </HStack>
        </VStack>
      </CardBox>
    </Center>
  );
}


