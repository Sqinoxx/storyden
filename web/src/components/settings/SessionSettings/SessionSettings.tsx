"use client";

import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { CardBox, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import {
  Props,
  SESSION_DURATION_OPTIONS,
  useSessionSettings,
} from "./useSessionSettings";

export function SessionSettings(props: Props) {
  const t = useTranslation();
  const { register, formState, onSubmit } = useSessionSettings(props);

  return (
    <styled.form
      width="full"
      display="flex"
      flexDirection="column"
      gap="4"
      onSubmit={onSubmit}
    >
      <CardBox className={lstack()} gap="8">
        <LStack>
          <Heading size="md">{t.sessionSettings.title}</Heading>
          <p>{t.sessionSettings.description}</p>
        </LStack>

        <FormControl>
          <FormLabel>{t.sessionSettings.durationLabel}</FormLabel>
          <styled.select
            width="full"
            maxW="64"
            height="9"
            px="2"
            borderWidth="thin"
            borderStyle="solid"
            borderColor="border.default"
            borderRadius="l2"
            bg="bg.default"
            color="fg.default"
            {...register("rememberMeDays")}
          >
            {SESSION_DURATION_OPTIONS.map((option) => (
              <option key={option.value} value={option.value}>
                {t.sessionSettings.durations[option.labelKey]}
              </option>
            ))}
          </styled.select>
          <FormHelperText>{t.sessionSettings.durationHelp}</FormHelperText>
        </FormControl>

        <WStack justifyContent="flex-end">
          <Button type="submit" loading={formState.isSubmitting}>
            {t.common.save}
          </Button>
        </WStack>
      </CardBox>
    </styled.form>
  );
}
