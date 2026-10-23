import { createListCollection } from "@ark-ui/react";

import { ColourField } from "@/components/form/ColourInput/ColourInput";

import { ContentFormField } from "@/components/content/ContentComposer/ContentField";
import { FormErrorText } from "@/components/ui/FormErrorText";
import { Button } from "@/components/ui/button";
import { DatePickerInputField } from "@/components/ui/form/DatePickerField";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { SelectField } from "@/components/ui/form/SelectField";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import {
  Box,
  CardBox,
  HStack,
  Stack,
  WStack,
  styled,
} from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { useTranslation } from "@/lib/i18n";

import { BannerEditor } from "./BannerEditor/BannerEditor";
import { IconEditor } from "./IconEditor/IconEditor";
import { LogoEditor } from "./LogoEditor/LogoEditor";
import { Form, Props, useBrandSettings } from "./useBrandSettings";

export function BrandSettingsForm(props: Props) {
  const t = useTranslation();
  const MOTD_TYPE_COLLECTION = createListCollection({
    items: [
      { label: t.settings.motd.typeCelebration, value: "celebration" },
      { label: t.settings.motd.typeInformation, value: "information" },
      { label: t.settings.motd.typeAlert, value: "alert" },
    ],
  });
  const {
    register,
    control,
    formState,
    onSubmit,
    currentIcon,
    onSaveIcon,
    onColourChangePreview,
    onClearMotdDates,
    onClearMotd,
    motdContentInitialValue,
    motdContentResetKey,
  } = useBrandSettings(props);

  return (
    <CardBox className={lstack()}>
      <styled.form
        width="full"
        display="flex"
        flexDirection="column"
        gap="4"
        alignItems="start"
        onSubmit={onSubmit}
      >
        <WStack>
          <Heading size="md">Brand settings</Heading>
          <Button type="submit">Save</Button>
        </WStack>

        <Stack
          gap="4"
          direction={{
            base: "column",
            lg: "row",
          }}
        >
          <FormControl>
            <FormLabel>Community name</FormLabel>
            <Input {...register("title")} />
            <FormHelperText>
              The name of your community. This appears in the sidebar, Google
              indexing and tab titles.
            </FormHelperText>
          </FormControl>
        </Stack>

        <FormControl display="flex" flexDirection="column">
          <FormLabel>Icon</FormLabel>

          <IconEditor initialValue={currentIcon} onSave={onSaveIcon} />

          <FormHelperText>
            Your icon will be automatically resized and optimised for various
            devices. It is used for the website favicon and a PWA app icon for
            iOS and Android devices.
          </FormHelperText>
        </FormControl>

        <FormControl display="flex" flexDirection="column">
          <FormLabel>Logo</FormLabel>

          <LogoEditor />
          <FormHelperText>
            Shown at the top of the home page. Upload separate images for light
            and dark mode. If only one is set, it is used for both.
          </FormHelperText>
        </FormControl>

        <FormControl display="flex" flexDirection="column">
          <FormLabel>Banner</FormLabel>

          <BannerEditor />
          <FormHelperText>
            Your banner will be used for link previews on other platforms.
          </FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>Description</FormLabel>
          <Input {...register("description")} />
          <FormHelperText>
            Describe your community with a few words here. This will be used for
            Google indexing, social previews and the PWA manifest.
          </FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>About</FormLabel>
          <CardBox>
            <ContentFormField
              control={control}
              name="content"
              // NOTE: Does not update if sidebar is changed. Doesn't matter...
              initialValue={props.settings.content}
              placeholder="About your community..."
            />
            <FormErrorText>{formState.errors.content?.message}</FormErrorText>
          </CardBox>
          <FormHelperText>
            You can write a longer description about your community here. You
            can use rich text formatting and include links and images.
          </FormHelperText>
        </FormControl>

        <FormControl>
          <FormLabel>Colour</FormLabel>
          <HStack>
            <Box>
              <ColourField
                name="accentColour"
                defaultValue={props.settings.accent_colour}
                control={control}
                onUpdate={onColourChangePreview}
              />
            </Box>
          </HStack>

          <FormHelperText>
            Pick a colour that best represents your community or brand. It will
            be used throughout the site for accenting certain elements such as
            buttons, mobile browser borders, PWA theme, etc.
          </FormHelperText>
        </FormControl>

        <WStack mt="2">
          <Heading size="sm">{t.settings.motd.title}</Heading>
          <Button type="submit" size="sm">
            {t.settings.motd.save}
          </Button>
        </WStack>

        <FormControl>
          <WStack alignItems="center">
            <FormLabel>{t.settings.motd.content}</FormLabel>
            <Button
              type="button"
              size="xs"
              variant="outline"
              onClick={onClearMotd}
            >
              {t.settings.motd.clear}
            </Button>
          </WStack>
          <CardBox>
            <ContentFormField
              control={control}
              name="motdContent"
              initialValue={motdContentInitialValue}
              resetKey={motdContentResetKey}
              placeholder={t.settings.motd.contentPlaceholder}
            />
            <FormErrorText>
              {formState.errors.motdContent?.message}
            </FormErrorText>
          </CardBox>
          <FormHelperText>{t.settings.motd.contentHelper}</FormHelperText>
        </FormControl>

        <Stack
          gap="4"
          direction={{
            base: "column",
            lg: "row",
          }}
          width="full"
        >
          <FormControl>
            <FormLabel>{t.settings.motd.startsAt}</FormLabel>
            <DatePickerInputField<Form> name="motdStartAt" control={control} />
            <FormErrorText>
              {formState.errors.motdStartAt?.message}
            </FormErrorText>
          </FormControl>

          <FormControl>
            <FormLabel>{t.settings.motd.endsAt}</FormLabel>
            <DatePickerInputField<Form> name="motdEndAt" control={control} />
            <FormErrorText>{formState.errors.motdEndAt?.message}</FormErrorText>
          </FormControl>
        </Stack>
        <Button
          type="button"
          size="xs"
          variant="outline"
          onClick={onClearMotdDates}
        >
          {t.settings.motd.clearDates}
        </Button>

        <FormControl>
          <FormLabel>{t.settings.motd.alertType}</FormLabel>
          <SelectField<Form, (typeof MOTD_TYPE_COLLECTION.items)[number]>
            control={control}
            name="motdType"
            collection={MOTD_TYPE_COLLECTION}
            placeholder={t.settings.motd.alertTypePlaceholder}
          />
          <FormErrorText>{formState.errors.motdType?.message}</FormErrorText>
          <FormHelperText>
            {t.settings.motd.alertTypeHelper}
          </FormHelperText>
        </FormControl>

        <WStack justifyContent="end">
          <Button type="submit">{t.settings.motd.save}</Button>
        </WStack>
      </styled.form>
    </CardBox>
  );
}
