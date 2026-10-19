import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { CardBox, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useMarketSettings } from "./useMarketSettings";

export function MarketSettingsForm(props: Props) {
  const t = useTranslation();
  const { control, formState, onSubmit } = useMarketSettings(props);

  return (
    <styled.form
      width="full"
      display="flex"
      flexDirection="column"
      gap="4"
      onSubmit={onSubmit}
    >
      <CardBox className={lstack()}>
        <WStack>
          <Heading size="md">{t.marketSettings.title}</Heading>
          <Button type="submit" loading={formState.isSubmitting}>
            {t.common.save}
          </Button>
        </WStack>

        <styled.p color="fg.muted">{t.marketSettings.description}</styled.p>

        <FormControl>
          <FormLabel>{t.marketSettings.durationLabel}</FormLabel>
          <NumberInputField
            control={control}
            name="listingDurationDays"
            scrubber={true}
            min={1}
            max={365}
            step={1}
          />
          <FormHelperText>{t.marketSettings.durationHelp}</FormHelperText>
        </FormControl>
      </CardBox>
    </styled.form>
  );
}
