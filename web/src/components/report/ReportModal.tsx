"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ReactNode } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { ModalDrawer } from "@/components/site/Modaldrawer/Modaldrawer";
import { UseDisclosureProps } from "@/utils/useDisclosure";

import { handle } from "@/api/client";
import { reportCreate } from "@/api/openapi-client/reports";
import { DatagraphItemKind } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Input } from "../ui/input";
import { useTranslation } from "@/lib/i18n";

const FormSchema = z.object({
  comment: z.string().max(1000).optional(),
});
type Form = z.infer<typeof FormSchema>;

export type ReportModalProps = UseDisclosureProps & {
  title: string;
  description: ReactNode;
  subject?: ReactNode;
  targetId: string;
  targetKind: DatagraphItemKind;
  submitLabel?: string;
  successMessage?: string;
  loadingMessage?: string;
};

export function ReportModal({
  title,
  description,
  subject,
  targetId,
  targetKind,
  submitLabel,
  successMessage,
  loadingMessage,
  ...disclosure
}: ReportModalProps) {
  const t = useTranslation();
  const defaultSubmitLabel = submitLabel || t.report.submitReport;
  const defaultSuccessMessage = successMessage || t.report.reportSubmitted;
  const defaultLoadingMessage = loadingMessage || t.report.sendingReport;
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: {
      comment: "",
    },
  });

  const handleSubmit = form.handleSubmit(async ({ comment }) => {
    await handle(
      async () => {
        await reportCreate({
          target_id: targetId,
          target_kind: targetKind,
          comment: comment?.trim() || undefined,
        });
      },
      {
        promiseToast: {
          loading: defaultLoadingMessage,
          success: defaultSuccessMessage,
        },
      },
    );

    form.reset();
    disclosure.onClose?.();
  });

  function handleCancel() {
    form.reset();
    disclosure.onClose?.();
  }

  return (
    <ModalDrawer title={title} {...disclosure}>
      <styled.form
        as="form"
        gap="4"
        alignItems="stretch"
        onSubmit={handleSubmit}
        className={lstack()}
      >
        <styled.div color="fg.muted">{description}</styled.div>

        {subject && (
          <styled.div
            borderWidth="thin"
            borderRadius="md"
            borderColor="border.subtle"
            background="bg.subtle"
            padding="3"
          >
            {subject}
          </styled.div>
        )}

        <FormControl>
          <FormLabel>{t.report.additionalDetails}</FormLabel>
          <Input
            {...form.register("comment")}
            placeholder={t.report.optionalContext}
            maxLength={1000}
            resize="vertical"
          />
          <FormHelperText>
            {t.report.helperText}
          </FormHelperText>
          <FormErrorText>
            {form.formState.errors.comment?.message}
          </FormErrorText>
        </FormControl>

        <WStack gap="2">
          <Button type="button" variant="outline" onClick={handleCancel}>
            {t.common.cancel}
          </Button>
          <Button
            type="submit"
            colorPalette="red"
            loading={form.formState.isSubmitting}
          >
            {defaultSubmitLabel}
          </Button>
        </WStack>
      </styled.form>
    </ModalDrawer>
  );
}
