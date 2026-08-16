"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

import { AccountCommonProps, AuthMode, Permission } from "@/api/openapi-schema";
import { resendVerificationEmail } from "@/api/verification";
import { Admonition } from "@/components/ui/admonition";
import { Button } from "@/components/ui/button";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { Settings } from "@/lib/settings/settings";
import { Box, HStack, LStack } from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";
import { toast } from "sonner";

type Props = {
  session: AccountCommonProps | undefined;
  settings: Settings;
};

const DISMISS_KEY = "storyden:verification-banner-dismissed";

export function VerificationBanner({ session, settings }: Props) {
  const { t } = useLanguage();
  // Dismissal is persisted so it does not reappear on every navigation.
  const [visible, setVisible] = useState(true);
  const [isSending, setSending] = useState(false);

  useEffect(() => {
    setVisible(window.localStorage.getItem(DISMISS_KEY) !== "true");
  }, []);

  if (!session) {
    return null;
  }

  const isAdmin = hasPermission(session, Permission.ADMINISTRATOR);

  const shouldShow =
    settings.authentication_mode === AuthMode.email &&
    session.verified_status === "none" &&
    !isAdmin;

  if (!shouldShow) {
    return null;
  }

  function handleDismiss(next: boolean) {
    setVisible(next);
    window.localStorage.setItem(DISMISS_KEY, next ? "false" : "true");
  }

  async function handleResend() {
    setSending(true);

    try {
      await toast.promise(resendVerificationEmail(), {
        loading: t.verification.resendLoading,
        success: t.verification.resendSuccess,
        error: t.verification.resendError,
      }).unwrap();
    } catch {
      // The toast already reported it.
    } finally {
      setSending(false);
    }
  }

  return (
    <Box mb="4">
      <Admonition
        value={visible}
        kind="failure"
        title={t.verification.title}
        onChange={handleDismiss}
      >
        <LStack gap="2">
          <p>{t.verification.body}</p>

          <HStack gap="2" flexWrap="wrap">
            <Button
              size="sm"
              variant="subtle"
              onClick={handleResend}
              loading={isSending}
            >
              {t.verification.resend}
            </Button>

            <Button size="sm" asChild>
              <Link href="/auth/verify/email?returnURL=/d">
                {t.verification.enterCode}
              </Link>
            </Button>
          </HStack>
        </LStack>
      </Admonition>
    </Box>
  );
}
