"use client";

import { useState } from "react";

import { Button, ButtonProps } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack } from "@/styled-system/jsx";

type Props = Omit<ButtonProps, "onClick"> & {
  confirmLabel: string;
  onConfirm: (reason: string) => Promise<void>;
  reasonLabel?: string;
  reasonPlaceholder?: string;
};

// Two-step button for decisions a member will be notified about, so a
// misclick on a phone does not decline or revoke anyone.
export function ConfirmButton({
  confirmLabel,
  onConfirm,
  reasonLabel,
  reasonPlaceholder,
  children,
  ...props
}: Props) {
  const t = useTranslation();
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [reason, setReason] = useState("");

  function cancel() {
    setConfirming(false);
    setReason("");
  }

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

  const buttons = (
    <HStack
      gap="1"
      flexWrap="wrap"
      flexBasis={{ base: "full", sm: "max" }}
      css={{ "& > *": { flex: { base: "1", sm: "initial" } } }}
    >
      <Button
        size="sm"
        variant="solid"
        colorPalette="red"
        loading={busy}
        aria-label={confirmLabel}
        onClick={async () => {
          setBusy(true);
          try {
            await onConfirm(reason.trim());
          } finally {
            setBusy(false);
            cancel();
          }
        }}
      >
        {confirmLabel}
      </Button>
      <Button size="sm" variant="ghost" onClick={cancel}>
        {t.common.cancel}
      </Button>
    </HStack>
  );

  if (!reasonLabel) return buttons;

  return (
    <LStack gap="1" flexBasis="full" w="full">
      <Input
        size="sm"
        fontSize={{ base: "md", md: "sm" }}
        aria-label={reasonLabel}
        placeholder={reasonPlaceholder ?? reasonLabel}
        value={reason}
        maxLength={500}
        autoFocus
        onChange={(e) => setReason(e.currentTarget.value)}
      />
      {buttons}
    </LStack>
  );
}
