import { Controller } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { IconButton } from "@/components/ui/icon-button";
import { AddIcon } from "@/components/ui/icons/Add";
import { ChevronDownIcon, ChevronUpIcon } from "@/components/ui/icons/Chevron";
import { DeleteIcon } from "@/components/ui/icons/Delete";
import { Input } from "@/components/ui/input";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { NumberInputField } from "@/components/ui/form/NumberInputField";
import { Heading } from "@/components/ui/heading";
import { Switch } from "@/components/ui/switch";
import { useTranslation } from "@/lib/i18n";
import { CardBox, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useMarketSettings } from "./useMarketSettings";

export function MarketSettingsForm(props: Props) {
  const t = useTranslation();
  const { control, register, formState, categories, onSubmit } =
    useMarketSettings(props);

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
          <Controller
            control={control}
            name="enabled"
            render={({ field }) => (
              <Switch
                checked={field.value}
                onCheckedChange={({ checked }) => field.onChange(checked)}
              >
                {t.marketSettings.enabledLabel}
              </Switch>
            )}
          />
          <FormHelperText>{t.marketSettings.enabledHelp}</FormHelperText>
        </FormControl>

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

        <FormControl>
          <FormLabel>{t.marketSettings.categoriesLabel}</FormLabel>
          <FormHelperText>{t.marketSettings.categoriesHelp}</FormHelperText>
          <LStack gap="2" mt="2">
            {categories.fields.map((field, index) => (
              <HStack key={field.key} w="full" gap="2">
                <Input
                  size="sm"
                  placeholder={t.marketSettings.categoryNamePlaceholder}
                  {...register(`categories.${index}.name`)}
                />
                {field.id && (
                  <styled.code
                    fontSize="xs"
                    color="fg.muted"
                    flexShrink="0"
                    title={t.marketSettings.categoryIdHelp}
                  >
                    {field.id}
                  </styled.code>
                )}
                <IconButton
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={index === 0}
                  aria-label={t.marketSettings.moveUp}
                  onClick={() => categories.move(index, index - 1)}
                >
                  <ChevronUpIcon />
                </IconButton>
                <IconButton
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={index === categories.fields.length - 1}
                  aria-label={t.marketSettings.moveDown}
                  onClick={() => categories.move(index, index + 1)}
                >
                  <ChevronDownIcon />
                </IconButton>
                <IconButton
                  type="button"
                  size="sm"
                  variant="ghost"
                  disabled={categories.fields.length <= 1}
                  aria-label={t.marketSettings.removeCategory}
                  onClick={() => categories.remove(index)}
                >
                  <DeleteIcon />
                </IconButton>
              </HStack>
            ))}
            <Button
              type="button"
              size="sm"
              variant="subtle"
              onClick={() => categories.append({ id: "", name: "" })}
            >
              <AddIcon /> {t.marketSettings.addCategory}
            </Button>
          </LStack>
          <FormErrorText>
            {formState.errors.categories?.message ??
              formState.errors.categories?.root?.message}
          </FormErrorText>
        </FormControl>
      </CardBox>
    </styled.form>
  );
}
