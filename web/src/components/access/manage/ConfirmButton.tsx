"use client";

import { useState } from "react";

import { Button, ButtonProps } from "@/components/ui/button";
import { useTranslation } from "@/lib/i18n";
import { HStack } from "@/styled-system/jsx";

type Props = Omit<ButtonProps, "onClick"> & {
  confirmLabel: string;
  onConfirm: () => Promise<void>;
};

// Two-step button for decisions a member will be notified about, so a
// misclick on a phone does not decline or revoke anyone.
export function ConfirmButton({
  confirmLabel,
  onConfirm,
  children,
  ...props
}: Props) {
  const t = useTranslation();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  if (!confirming) {
    return (
      <Button
        size="sm"
        variant="ghost"
        {...props}
        onClick={() => setConfirming(true)}
      >
        {children}
      </Button>
    );
  }

  return (
    <HStack gap="1" flexWrap="wrap">
      <Button
        size="sm"
        variant="solid"
        bg="bg.destructive"
        loading={busy}
        aria-label={confirmLabel}
        onClick={async () => {
          setBusy(true);
          try {
            await onConfirm();
          } finally {
            setBusy(false);
            setConfirming(false);
          }
        }}
      >
        {confirmLabel}
      </Button>
      <Button size="sm" variant="ghost" onClick={() => setConfirming(false)}>
        {t.common.cancel}
      </Button>
    </HStack>
  );
}
