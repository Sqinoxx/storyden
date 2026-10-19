"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import {
  accountPhoneRemove,
  accountPhoneSet,
  useAccountPhoneGet,
} from "@/api/openapi-client/accounts";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { CardBox, HStack, LStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

const FormSchema = z.object({
  phone: z
    .string()
    .trim()
    .min(4)
    .max(32)
    .regex(/^[+0-9][0-9 ()/.-]*$/),
});
type Form = z.infer<typeof FormSchema>;

export function PhoneSettings() {
  const t = useTranslation();
  const { data, mutate } = useAccountPhoneGet();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: { phone: "" },
  });

  useEffect(() => {
    if (data?.phone) {
      form.reset({ phone: data.phone });
    }
  }, [data?.phone, form]);

  const onSubmit = form.handleSubmit(async ({ phone }) => {
    await handle(
      async () => {
        const saved = await accountPhoneSet({ phone });
        await mutate(saved, { revalidate: false });
        form.reset({ phone: saved.phone ?? "" });
      },
      {
        promiseToast: {
          loading: t.phoneSettings.saving,
          success: t.phoneSettings.saved,
        },
      },
    );
  });

  async function handleRemove() {
    await handle(
      async () => {
        const cleared = await accountPhoneRemove();
        await mutate(cleared, { revalidate: false });
        form.reset({ phone: "" });
      },
      {
        promiseToast: {
          loading: t.phoneSettings.saving,
          success: t.phoneSettings.removed,
        },
      },
    );
  }

  return (
    <styled.form width="full" onSubmit={onSubmit} id="phone">
      <CardBox className={lstack()} gap="4">
        <LStack>
          <Heading size="md">{t.phoneSettings.title}</Heading>
          <styled.p color="fg.muted">{t.phoneSettings.description}</styled.p>
        </LStack>

        <FormControl>
          <FormLabel>{t.phoneSettings.label}</FormLabel>
          <Input
            type="tel"
            autoComplete="tel"
            maxW="72"
            placeholder={t.phoneSettings.placeholder}
            {...form.register("phone")}
          />
          <FormHelperText>{t.phoneSettings.help}</FormHelperText>
          <FormErrorText>{form.formState.errors.phone?.message}</FormErrorText>
        </FormControl>

        <HStack justifyContent="flex-end">
          {data?.phone && (
            <Button type="button" variant="ghost" onClick={handleRemove}>
              {t.phoneSettings.remove}
            </Button>
          )}
          <Button type="submit" loading={form.formState.isSubmitting}>
            {t.common.save}
          </Button>
        </HStack>
      </CardBox>
    </styled.form>
  );
}
