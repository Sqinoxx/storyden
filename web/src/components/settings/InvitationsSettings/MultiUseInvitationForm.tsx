"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useSWRConfig } from "swr";
import { z } from "zod";

import { handle } from "@/api/client";
import {
  getInvitationListKey,
  invitationCreate,
} from "@/api/openapi-client/invitations";
import { Identifier } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { CancelIcon } from "@/components/ui/icons/Cancel";
import { SaveIcon } from "@/components/ui/icons/Save";
import { useTranslation } from "@/lib/i18n";
import { CardBox, Flex, HStack, styled } from "@/styled-system/jsx";

export const INVITATION_EXPIRY_OPTIONS = [
  { value: "never", labelKey: "never" },
  { value: "1", labelKey: "oneDay" },
  { value: "7", labelKey: "sevenDays" },
  { value: "30", labelKey: "thirtyDays" },
] as const;

export const FormSchema = z.object({
  maxUses: z.number().int().min(2).max(100_000),
  expiry: z.enum(["never", "1", "7", "30"]),
});
export type Form = z.infer<typeof FormSchema>;

type Props = {
  onSuccess: (id: Identifier) => void;
  onCancel: () => void;
};

export function MultiUseInvitationForm(props: Props) {
  const t = useTranslation();
  const { mutate } = useSWRConfig();
  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: { maxUses: 5, expiry: "7" },
  });

  const handleSubmit = form.handleSubmit(async ({ maxUses, expiry }) => {
    const days = expiry === "never" ? undefined : Number(expiry);
    const expiresAt = days
      ? new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString()
      : undefined;

    await handle(
      async () => {
        const created = await invitationCreate({
          max_uses: maxUses,
          expires_at: expiresAt,
        });
        props.onSuccess(created.id);
      },
      {
        cleanup: async () => {
          await mutate(getInvitationListKey());
        },
      },
    );
  });

  return (
    <CardBox>
      <styled.form
        display="flex"
        flexDirection="column"
        gap="4"
        onSubmit={handleSubmit}
      >
        <Flex
          flexDir={{ base: "column", md: "row" }}
          gap="4"
          alignItems="flex-start"
        >
          <FormControl>
            <FormLabel>{t.invitations.multiUsesLabel}</FormLabel>
            <NumberInputField<Form>
              control={form.control}
              name="maxUses"
              min={2}
              max={100_000}
              step={1}
              scrubber
            />
            <FormHelperText>{t.invitations.multiUsesHelp}</FormHelperText>
            <FormErrorText>
              {form.formState.errors.maxUses?.message}
            </FormErrorText>
          </FormControl>

          <FormControl>
            <FormLabel>{t.invitations.expiryLabel}</FormLabel>
            <styled.select
              width="full"
              height="9"
              px="2"
              borderWidth="thin"
              borderStyle="solid"
              borderColor="border.default"
              borderRadius="l2"
              bg="bg.default"
              color="fg.default"
              {...form.register("expiry")}
            >
              {INVITATION_EXPIRY_OPTIONS.map((option) => (
                <option key={option.value} value={option.value}>
                  {t.invitations.expiryOptions[option.labelKey]}
                </option>
              ))}
            </styled.select>
          </FormControl>
        </Flex>

        <HStack gap="0" alignSelf="flex-end">
          <Button
            size="sm"
            variant="solid"
            borderRightRadius="none"
            loading={form.formState.isSubmitting}
          >
            <SaveIcon /> {t.actions.create}
          </Button>
          <Button
            type="button"
            size="sm"
            variant="subtle"
            borderLeftRadius="none"
            onClick={props.onCancel}
          >
            <CancelIcon /> {t.common.cancel}
          </Button>
        </HStack>
      </styled.form>
    </CardBox>
  );
}
