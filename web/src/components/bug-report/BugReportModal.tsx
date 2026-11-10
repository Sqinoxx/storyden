"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ChangeEvent, useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { toast } from "sonner";
import { z } from "zod";

import { handle } from "@/api/client";
import { bugReportCreate } from "@/api/openapi-client/bug-reports";
import { Asset, BugReportCategory } from "@/api/openapi-schema";
import { useImageUpload } from "@/components/content/useImageUpload";
import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";
import { getAssetURL } from "@/utils/asset";
import { deriveError } from "@/utils/error";
import { UseDisclosureProps } from "@/utils/useDisclosure";

const MAX_DESCRIPTION_LENGTH = 5000;

const CATEGORIES = [BugReportCategory.bug, BugReportCategory.other] as const;

type Variant = "bug" | "suggestion";

const FormSchema = z.object({
  category: z.nativeEnum(BugReportCategory),
  description: z.string().trim().min(1).max(MAX_DESCRIPTION_LENGTH),
});
type Form = z.infer<typeof FormSchema>;

type Props = UseDisclosureProps & {
  variant?: Variant;
};

export function BugReportModal({ variant = "bug", ...disclosure }: Props) {
  const t = useTranslation();
  const isSuggestion = variant === "suggestion";
  const copy = isSuggestion ? t.suggestion : t.bugReport;
  const defaultValues: Form = {
    category: isSuggestion
      ? BugReportCategory.suggestion
      : BugReportCategory.bug,
    description: "",
  };
  const { upload } = useImageUpload();
  const fileInputRef = useRef<HTMLInputElement>(null);
  const [screenshot, setScreenshot] = useState<Asset | null>(null);
  const [uploading, setUploading] = useState(false);

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues,
  });

  function reset() {
    form.reset(defaultValues);
    setScreenshot(null);
  }

  async function handleFileChange(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;

    if (!file.type.startsWith("image/")) {
      toast.error(t.bugReport.onlyImages);
      return;
    }

    setUploading(true);
    try {
      setScreenshot(await upload(file));
    } catch (error) {
      toast.error(deriveError(error));
    } finally {
      setUploading(false);
    }
  }

  const handleSubmit = form.handleSubmit(async ({ category, description }) => {
    try {
      await handle(
        () =>
          bugReportCreate({
            category,
            description,
            page_url: window.location.href,
            user_agent: navigator.userAgent,
            asset_id: screenshot?.id,
          }),
        {
          promiseToast: {
            loading: copy.sending,
            success: copy.sent,
          },
        },
      );
    } catch {
      return;
    }

    reset();
    disclosure.onClose?.();
  });

  function handleCancel() {
    reset();
    disclosure.onClose?.();
  }

  return (
    <ModalDrawer title={copy.title} {...disclosure}>
      <styled.form
        gap="4"
        alignItems="stretch"
        onSubmit={handleSubmit}
        className={lstack()}
      >
        <styled.p color="fg.muted">{copy.description}</styled.p>

        {isSuggestion ? (
          <LinkButton
            href="/suggestions"
            size="xs"
            variant="subtle"
            w="fit"
            onClick={handleCancel}
          >
            {t.suggestion.mySuggestions}
          </LinkButton>
        ) : (
          <FormControl>
            <FormLabel>{t.bugReport.category}</FormLabel>
            <Controller
              control={form.control}
              name="category"
              render={({ field }) => (
                <HStack gap="2" flexWrap="wrap">
                  {CATEGORIES.map((c) => (
                    <Button
                      key={c}
                      type="button"
                      size="sm"
                      variant={field.value === c ? "solid" : "outline"}
                      onClick={() => field.onChange(c)}
                    >
                      {t.bugReport.categories[c]}
                    </Button>
                  ))}
                </HStack>
              )}
            />
          </FormControl>
        )}

        <FormControl>
          <FormLabel>{copy.descriptionLabel}</FormLabel>
          <styled.textarea
            {...form.register("description")}
            placeholder={copy.descriptionPlaceholder}
            maxLength={MAX_DESCRIPTION_LENGTH}
            minH="32"
            w="full"
            resize="vertical"
            borderWidth="thin"
            borderColor="border.default"
            borderRadius="l2"
            p="3"
            bgColor="bg.default"
            color="fg.default"
            fontSize="sm"
          />
          <FormErrorText>
            {form.formState.errors.description && copy.descriptionRequired}
          </FormErrorText>
        </FormControl>

        <FormControl>
          <FormLabel>{t.bugReport.screenshot}</FormLabel>
          <input
            ref={fileInputRef}
            type="file"
            accept="image/*"
            hidden
            onChange={handleFileChange}
          />
          {screenshot ? (
            <HStack gap="3" alignItems="center">
              <styled.img
                src={getAssetURL(screenshot.path)}
                alt=""
                maxH="24"
                maxW="40"
                objectFit="contain"
                borderWidth="thin"
                borderColor="border.subtle"
                borderRadius="md"
              />
              <Button
                type="button"
                size="xs"
                variant="subtle"
                onClick={() => setScreenshot(null)}
              >
                {t.bugReport.removeScreenshot}
              </Button>
            </HStack>
          ) : (
            <Button
              type="button"
              size="sm"
              variant="outline"
              loading={uploading}
              onClick={() => fileInputRef.current?.click()}
            >
              {uploading
                ? t.bugReport.uploadingScreenshot
                : t.bugReport.addScreenshot}
            </Button>
          )}
          <FormHelperText>{t.bugReport.pageInfo}</FormHelperText>
        </FormControl>

        <WStack gap="2">
          <Button type="button" variant="outline" onClick={handleCancel}>
            {t.common.cancel}
          </Button>
          <Button
            type="submit"
            loading={form.formState.isSubmitting}
            disabled={uploading}
          >
            {copy.submit}
          </Button>
        </WStack>
      </styled.form>
    </ModalDrawer>
  );
}
