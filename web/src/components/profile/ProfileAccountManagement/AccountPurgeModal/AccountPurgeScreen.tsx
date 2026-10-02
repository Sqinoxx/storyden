import { ModerationActionPurgeAccountContentType } from "@/api/openapi-schema";
import { FormControl } from "@/components/ui/FormControl";
import { FormLabel } from "@/components/ui/FormLabel";
import { Button } from "@/components/ui/button";
import { CardGroupSelect } from "@/components/ui/form/CardGroupSelect";
import { WarningIcon } from "@/components/ui/icons/Warning";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useAccountPurgeScreen } from "./useAccountPurge";

export function AccountPurgeScreen(props: Props) {
  const t = useTranslation();
  const {
    form,
    handlers: { handlePurge },
  } = useAccountPurgeScreen(props);

  const contentTypes = Object.values(
    ModerationActionPurgeAccountContentType,
  ).map((value) => ({
    value,
    label: t.moderation.purgeTypes[value].name,
    description: t.moderation.purgeTypes[value].description,
  }));

  return (
    <styled.form
      className={lstack()}
      h="full"
      justifyContent="space-between"
      onSubmit={handlePurge}
    >
      <LStack px="0.5" maxH="full" pb="1" overflowY="scroll">
        <Box
          p="3"
          borderRadius="sm"
          borderWidth="thin"
          borderColor="border.warning"
          bgColor="bg.warning"
        >
          <HStack gap="2" color="fg.warning">
            <WarningIcon w="5" flexShrink="0" />
            <LStack gap="1">
              <styled.p fontWeight="semibold" fontSize="sm">
                {t.moderation.destructiveAction}
              </styled.p>
              <styled.p fontSize="xs">{t.moderation.purgeWarning}</styled.p>
            </LStack>
          </HStack>
        </Box>

        <FormControl>
          <FormLabel>{t.moderation.purgeContentTypes}</FormLabel>
          <CardGroupSelect
            control={form.control}
            name="contentTypes"
            items={contentTypes}
          />
          <styled.p fontSize="xs" color="fg.subtle" mt="1">
            {t.moderation.purgeSelectHint}
          </styled.p>
        </FormControl>
      </LStack>

      <WStack>
        <Button
          flexGrow="1"
          variant="solid"
          disabled={!form.formState.isDirty || form.formState.isSubmitting}
          loading={form.formState.isSubmitting}
          type="submit"
        >
          {t.moderation.purgeSubmit}
        </Button>
      </WStack>
    </styled.form>
  );
}
