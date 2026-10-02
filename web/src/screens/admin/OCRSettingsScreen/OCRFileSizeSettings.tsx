"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useAdminSettingsGet } from "@/api/openapi-client/admin";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { useSettingsMutation } from "@/lib/settings/mutation";
import { Box, HStack, styled } from "@/styled-system/jsx";

const DEFAULT_OCR_MAX_FILE_SIZE_MB = 10;
const DEFAULT_OCR_TIMEOUT_SECONDS = 60;

const FormSchema = z.object({
  ocrMaxFileSizeMb: z.number().min(1).max(1024),
  ocrTimeoutSeconds: z.number().min(5).max(3600),
});
type Form = z.infer<typeof FormSchema>;

export function OCRFileSizeSettings({ onSaved }: { onSaved?: () => void }) {
  const { data } = useAdminSettingsGet();
  if (!data) return null;

  return (
    <OCRFileSizeForm
      initial={{
        ocrMaxFileSizeMb:
          data.services?.assets?.ocr_max_file_size_mb ??
          DEFAULT_OCR_MAX_FILE_SIZE_MB,
        ocrTimeoutSeconds:
          data.services?.assets?.ocr_timeout_seconds ??
          DEFAULT_OCR_TIMEOUT_SECONDS,
      }}
      onSaved={onSaved}
    />
  );
}

function OCRFileSizeForm({
  initial,
  onSaved,
}: {
  initial: Form;
  onSaved?: () => void;
}) {
  const { revalidate, updateSettings } = useSettingsMutation();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: initial,
  });

  const onSubmit = form.handleSubmit(async (data) => {
    await handle(
      async () => {
        await updateSettings({
          services: {
            assets: {
              ocr_max_file_size_mb: data.ocrMaxFileSizeMb,
              ocr_timeout_seconds: data.ocrTimeoutSeconds,
            },
          },
        });
        onSaved?.();
      },
      {
        promiseToast: {
          loading: "Speichere…",
          success: "Gespeichert",
        },
        cleanup: async () => {
          await revalidate();
        },
      },
    );
  });

  return (
    <styled.form onSubmit={onSubmit}>
      <Box
        p="5"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.default"
      >
        <HStack justifyContent="space-between" mb="2">
          <styled.h3 fontSize="md" fontWeight="bold" color="fg.default">
            Einstellungen
          </styled.h3>
          <Button type="submit" size="sm" loading={form.formState.isSubmitting}>
            Speichern
          </Button>
        </HStack>

        <FormControl>
          <FormLabel>Maximale Dateigröße für OCR (MB)</FormLabel>
          <NumberInputField
            control={form.control}
            name="ocrMaxFileSizeMb"
            scrubber={true}
            min={1}
            max={1024}
            step={5}
          />
          <FormHelperText>
            Größere Dateien werden bei der Texterkennung übersprungen. Bereits
            übersprungene Dateien werden erst nach „OCR erneut ausführen“ neu
            verarbeitet. Große, gescannte PDFs können das Zeitlimit
            überschreiten und belasten den Server stärker.
          </FormHelperText>
        </FormControl>

        <FormControl mt="4">
          <FormLabel>Zeitlimit pro Datei (Sekunden)</FormLabel>
          <NumberInputField
            control={form.control}
            name="ocrTimeoutSeconds"
            scrubber={true}
            min={5}
            max={3600}
            step={30}
          />
          <FormHelperText>
            Dauert die Texterkennung einer Datei länger, wird sie abgebrochen
            und übersprungen. Höhere Werte helfen bei großen, gescannten PDFs,
            blockieren aber währenddessen alle anderen Dateien, da immer nur
            eine Datei gleichzeitig verarbeitet wird.
          </FormHelperText>
        </FormControl>
      </Box>
    </styled.form>
  );
}
