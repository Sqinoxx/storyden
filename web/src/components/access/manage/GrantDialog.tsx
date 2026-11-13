"use client";

import { useMemo, useState } from "react";
import { toast } from "sonner";

import { handle } from "@/api/client";
import { accessGrantSet } from "@/api/openapi-client/access";
import { AccessPass } from "@/api/openapi-schema";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { FormControl } from "@/components/ui/FormControl";
import { FormLabel } from "@/components/ui/FormLabel";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  ExpiryPreset,
  dateInputValue,
  endOfDayFromInput,
  expiryPresets,
} from "@/lib/access/expiry";
import { useAccessRevalidate } from "@/lib/access/useAccessRevalidate";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";

import { MemberPicker } from "./MemberPicker";

type Choice = ExpiryPreset["key"] | "custom";

export type GrantDialogProps = {
  isOpen: boolean;
  onClose: () => void;
  passes: AccessPass[];
  passID?: string;
  memberHandle?: string;
  memberName?: string;
  note?: string;
  defaultChoice?: Choice;
};

export function GrantDialog(props: GrantDialogProps) {
  if (!props.isOpen) return null;
  return <GrantDialogContent {...props} />;
}

function GrantDialogContent({
  isOpen,
  onClose,
  passes,
  passID,
  memberHandle,
  memberName,
  note: initialNote,
  defaultChoice = "thisTerm",
}: GrantDialogProps) {
  const t = useTranslation();
  const revalidate = useAccessRevalidate();
  const presets = useMemo(() => expiryPresets(new Date()), []);

  const [selectedPass, setSelectedPass] = useState(
    passID ?? passes[0]?.id ?? "",
  );
  const [handleInput, setHandleInput] = useState(memberHandle ?? "");
  const [choice, setChoice] = useState<Choice>(defaultChoice);
  const [customDate, setCustomDate] = useState(() =>
    dateInputValue(presets[1]?.expiresAt ?? new Date()),
  );
  const [note, setNote] = useState(initialNote ?? "");
  const [busy, setBusy] = useState(false);

  const expiresAt = (() => {
    if (choice === "custom") return endOfDayFromInput(customDate);
    return presets.find((p) => p.key === choice)?.expiresAt;
  })();

  const canSubmit =
    selectedPass !== "" &&
    handleInput.trim() !== "" &&
    (choice !== "custom" || expiresAt !== undefined);

  const title = t.access.manage.grantTitle.replace(
    "{name}",
    memberName ?? (handleInput || t.access.manage.memberHandle),
  );

  async function onSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!canSubmit) return;

    setBusy(true);
    await handle(
      async () => {
        await accessGrantSet(
          selectedPass,
          handleInput.trim().replace(/^@/, ""),
          {
            expires_at: expiresAt?.toISOString(),
            note: note.trim() || undefined,
          },
        );
        await revalidate();
        toast.success(t.access.manage.granted);
        onClose();
      },
      { cleanup: async () => setBusy(false) },
    );
  }

  return (
    <ModalDrawer isOpen={isOpen} onClose={onClose} title={title}>
      <styled.form
        display="flex"
        flexDir="column"
        gap="4"
        w="full"
        onSubmit={onSubmit}
        data-testid="access-grant-dialog"
      >
        {!memberHandle && (
          <FormControl>
            <FormLabel>{t.access.manage.memberSearch}</FormLabel>
            <MemberPicker onChange={setHandleInput} />
          </FormControl>
        )}

        {!passID && (
          <FormControl>
            <FormLabel>{t.access.manage.pass}</FormLabel>
            <styled.select
              id="access-grant-pass"
              aria-label={t.access.manage.pass}
              value={selectedPass}
              onChange={(e) => setSelectedPass(e.currentTarget.value)}
              borderWidth="thin"
              borderColor="border.default"
              borderRadius="sm"
              p="2"
              bg="bg.default"
            >
              {passes.map((p) => (
                <option key={p.id} value={p.id}>
                  {p.name}
                </option>
              ))}
            </styled.select>
          </FormControl>
        )}

        <FormControl>
          <FormLabel>{t.access.manage.validity}</FormLabel>
          <HStack gap="2" flexWrap="wrap" role="radiogroup">
            {presets.map((p) => (
              <ChoiceButton
                key={p.key}
                active={choice === p.key}
                onClick={() => setChoice(p.key)}
              >
                {p.key === "none"
                  ? t.access.manage.forever
                  : t.access.manage.endOfTerm.replace("{term}", p.label ?? "")}
              </ChoiceButton>
            ))}
            <ChoiceButton
              active={choice === "custom"}
              onClick={() => setChoice("custom")}
            >
              {t.access.manage.customDate}
            </ChoiceButton>
          </HStack>
          {choice === "custom" && (
            <Input
              type="date"
              aria-label={t.access.manage.validity}
              value={customDate}
              min={dateInputValue(new Date())}
              onChange={(e) => setCustomDate(e.currentTarget.value)}
              mt="2"
            />
          )}
        </FormControl>

        <FormControl>
          <FormLabel>{t.access.manage.note}</FormLabel>
          <styled.textarea
            id="access-grant-note"
            aria-label={t.access.manage.note}
            rows={2}
            value={note}
            onChange={(e) => setNote(e.currentTarget.value)}
            placeholder={t.access.manage.notePlaceholder}
            width="full"
            borderWidth="thin"
            borderRadius="sm"
            borderColor="border.default"
            padding="2"
          />
        </FormControl>

        <LStack>
          <Button type="submit" w="full" loading={busy} disabled={!canSubmit}>
            {t.access.manage.confirmGrant}
          </Button>
        </LStack>
      </styled.form>
    </ModalDrawer>
  );
}

function ChoiceButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Button
      type="button"
      size="sm"
      variant={active ? "solid" : "outline"}
      role="radio"
      aria-checked={active}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}
