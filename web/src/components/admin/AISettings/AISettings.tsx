import { Control, Controller } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { Switch } from "@/components/ui/switch";
import { useTranslation } from "@/lib/i18n";
import { CardBox, HStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Form, Props, useAISettings } from "./useAISettings";

type ToggleName = Exclude<keyof Form, "tagMeanings">;

function Toggle(props: {
  control: Control<Form>;
  name: ToggleName;
  label: string;
  help: string;
  disabled?: boolean;
}) {
  return (
    <FormControl>
      <Controller
        control={props.control}
        name={props.name}
        render={({ field }) => (
          <Switch
            checked={field.value}
            disabled={props.disabled}
            onCheckedChange={({ checked }) => field.onChange(checked)}
          >
            {props.label}
          </Switch>
        )}
      />
      <FormHelperText>{props.help}</FormHelperText>
    </FormControl>
  );
}

export function AISettingsForm(props: Props) {
  const t = useTranslation();
  const {
    configured,
    control,
    register,
    formState,
    generating,
    draftResult,
    generateMeanings,
    onSubmit,
  } = useAISettings(props);

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
          <Heading size="md">{t.aiSettings.title}</Heading>
          <Button type="submit" loading={formState.isSubmitting}>
            {t.common.save}
          </Button>
        </WStack>

        <styled.p color="fg.muted">{t.aiSettings.description}</styled.p>

        {!configured && (
          <styled.p
            p="3"
            borderRadius="md"
            bg="bg.warning"
            color="fg.warning"
            fontSize="sm"
          >
            {t.aiSettings.notConfigured}
          </styled.p>
        )}

        <Toggle
          control={control}
          name="threadSummary"
          label={t.aiSettings.threadSummaryLabel}
          help={t.aiSettings.threadSummaryHelp}
          disabled={!configured}
        />
        <Toggle
          control={control}
          name="tagSuggestion"
          label={t.aiSettings.tagSuggestionLabel}
          help={t.aiSettings.tagSuggestionHelp}
          disabled={!configured}
        />
        <Toggle
          control={control}
          name="semesterSuggestion"
          label={t.aiSettings.semesterSuggestionLabel}
          help={t.aiSettings.semesterSuggestionHelp}
        />
        <Toggle
          control={control}
          name="libraryAssist"
          label={t.aiSettings.libraryAssistLabel}
          help={t.aiSettings.libraryAssistHelp}
          disabled={!configured}
        />

        <FormControl>
          <WStack alignItems="end">
            <FormLabel>{t.aiSettings.tagMeaningsLabel}</FormLabel>
            <Button
              type="button"
              size="sm"
              variant="subtle"
              loading={generating}
              disabled={!configured}
              onClick={generateMeanings}
            >
              <IntelligenceIcon /> {t.aiSettings.tagMeaningsGenerate}
            </Button>
          </WStack>
          <styled.textarea
            {...register("tagMeanings")}
            width="full"
            borderWidth="thin"
            borderRadius="sm"
            borderColor="border.default"
            padding="2"
            fontFamily="mono"
            fontSize="sm"
            rows={10}
            placeholder={
              "endo: endodontics (root canal treatment)\nkfo: orthodontics (braces, brackets, aligners)"
            }
          />
          {generating && (
            <HStack color="fg.muted" fontSize="sm">
              {t.aiSettings.tagMeaningsGenerating}
            </HStack>
          )}
          {draftResult && (
            <styled.p fontSize="sm" color="fg.default">
              {draftResult}
            </styled.p>
          )}
          <FormHelperText>{t.aiSettings.tagMeaningsHelp}</FormHelperText>
        </FormControl>
      </CardBox>
    </styled.form>
  );
}
