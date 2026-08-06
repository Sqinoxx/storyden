"use client";

import { useState } from "react";
import { useSWRConfig } from "swr";

import { fetcher, handle } from "@/api/client";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import { useLanguage } from "@/lib/i18n/LanguageContext";
import { css } from "@/styled-system/css";
import { Box, CardBox, HStack, LStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

export function AccountSettings() {
  const { t } = useLanguage();
  const { mutate } = useSWRConfig();
  const [isOpen, setIsOpen] = useState(false);
  const [confirmInput, setConfirmInput] = useState("");
  const [isDeleting, setIsDeleting] = useState(false);

  const confirmWord = "LÖSCHEN";
  const isConfirmed = confirmInput.trim().toUpperCase() === confirmWord;

  async function handleDeleteAccount() {
    if (!isConfirmed || isDeleting) return;

    setIsDeleting(true);
    const res = await handle(
      async () => {
        return await fetcher<void>({
          url: "/accounts",
          method: "DELETE",
        });
      },
      {
        promiseToast: {
          loading: t.settings?.deletingAccount || "Konto wird gelöscht...",
          success: t.settings?.accountDeletedSuccess || "Dein Konto wurde erfolgreich gelöscht.",
        },
        cleanup: async () => {
          setIsDeleting(false);
        },
      },
    );

    if (res !== undefined) {
      setIsOpen(false);
      mutate(() => true, undefined, { revalidate: false });
      window.location.href = "/";
    }
  }

  return (
    <>
      <CardBox className={lstack()} gap="6">
        <LStack gap="1">
          <Heading size="md">
            {t.settings?.accountTitle || "Konto-Einstellungen"}
          </Heading>
          <styled.p color="fg.muted">
            {t.settings?.accountSubtitle || "Verwalte deine Kontoeinstellungen und deinen Kontostatus."}
          </styled.p>
        </LStack>

        <div className={css({ borderTopWidth: "thin", borderColor: "border.subtle", pt: "6" })}>
          <LStack gap="4">
            <div
              className={css({
                p: "4",
                borderRadius: "l2",
                borderWidth: "thin",
                borderColor: "red.8",
                bg: "red.1",
                _dark: { bg: "red.12", borderColor: "red.10" },
              })}
            >

              <LStack gap="3">
                <Heading size="sm" className={css({ color: "red.10", _dark: { color: "red.8" } })}>
                  {t.settings?.dangerZoneTitle || "Gefahrenzone: Konto löschen"}
                </Heading>
                <styled.p color="fg.muted">
                  {t.settings?.dangerZoneDescription ||
                    "Wenn du dein Konto löschst, werden deine persönlichen Anmeldedaten, E-Mails und Sitzungen unwiderruflich entfernt. Deine verfassten Beiträge bleiben anonymisiert erhalten."}
                </styled.p>

                <div className={css({ alignSelf: "flex-start" })}>
                  <Button
                    variant="solid"
                    className={css({
                      bg: "red.9",
                      color: "white",
                      _hover: { bg: "red.10" },
                    })}
                    onClick={() => setIsOpen(true)}
                  >
                    {t.settings?.deleteAccountButton || "Konto löschen..."}
                  </Button>
                </div>
              </LStack>
            </div>
          </LStack>
        </div>
      </CardBox>

      {/* Confirmation Dialog Overlay */}
      {isOpen && (
        <div
          style={{
            position: "fixed",
            top: 0,
            left: 0,
            right: 0,
            bottom: 0,
            zIndex: 100,
            backgroundColor: "rgba(0, 0, 0, 0.75)",
            backdropFilter: "blur(4px)",
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            padding: "16px",
          }}
          onClick={() => setIsOpen(false)}
        >
          <div
            className={css({
              bg: "bg.muted",
              borderRadius: "l3",
              p: "6",
            })}
            style={{
              maxWidth: "500px",
              width: "100%",
            }}
            onClick={(e) => e.stopPropagation()}
          >
            <LStack gap="5">
              <Box>
                <Heading size="md" className={css({ color: "red.10", _dark: { color: "red.8" } })}>
                  {t.settings?.confirmDeleteTitle || "Konto unwiderruflich löschen?"}
                </Heading>
                <styled.p color="fg.muted" className={css({ mt: "2" })}>
                  {t.settings?.confirmDeleteDescription ||
                    "Diese Aktion kann nicht rückgängig gemacht werden. Dein Profil wird anonymisiert und deine Zugangsdaten gelöscht."}
                </styled.p>
              </Box>

              <FormControl>
                <FormLabel>
                  {t.settings?.confirmDeleteInputLabel || `Gib zur Bestätigung "${confirmWord}" ein:`}
                </FormLabel>
                <Input
                  type="text"
                  placeholder={confirmWord}
                  value={confirmInput}
                  onChange={(e) => setConfirmInput(e.target.value)}
                  autoFocus
                />
                <FormHelperText>
                  {t.settings?.confirmDeleteInputHelper || `Tippe "${confirmWord}" in Großbuchstaben ein.`}
                </FormHelperText>
              </FormControl>

              <HStack className={css({ justifyContent: "flex-end", gap: "3", mt: "2" })}>
                <Button
                  variant="ghost"
                  onClick={() => {
                    setIsOpen(false);
                    setConfirmInput("");
                  }}
                  disabled={isDeleting}
                >
                  Abbrechen
                </Button>
                <Button
                  variant="solid"
                  className={css({
                    bg: "red.9",
                    color: "white",
                    _hover: { bg: "red.10" },
                  })}
                  onClick={handleDeleteAccount}
                  disabled={!isConfirmed || isDeleting}
                >
                  {isDeleting
                    ? (t.settings?.deletingAccount || "Löschen...")
                    : (t.settings?.confirmDeleteButton || "Konto endgültig löschen")}
                </Button>
              </HStack>
            </LStack>
          </div>
        </div>
      )}
    </>
  );
}
