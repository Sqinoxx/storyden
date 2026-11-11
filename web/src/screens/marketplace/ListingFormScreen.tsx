"use client";

import { PropsWithChildren } from "react";

import { useAccountPhoneGet } from "@/api/openapi-client/accounts";
import { ListingPriceType } from "@/api/openapi-schema";
import { blockSubmitWhileUploading } from "@/components/content/blockSubmitWhileUploading";
import {
  ListingImagesField,
  MAX_LISTING_IMAGES,
} from "@/components/marketplace/ListingImagesField";
import { NativeSelect } from "@/components/marketplace/NativeSelect";
import { PhoneRequiredNotice } from "@/components/marketplace/PhoneRequiredNotice";
import {
  conditionOptions,
  priceTypeOptions,
} from "@/components/marketplace/options";
import { useMarketCategories } from "@/components/marketplace/useMarketCategories";
import { UnreadyBanner } from "@/components/site/Unready";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Spinner } from "@/components/ui/Spinner";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { Input, InputPrefix } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import {
  addDays,
  formatDate,
  formatRemaining,
} from "@/lib/marketplace/marketplace";
import { useSettings } from "@/lib/settings/settings-client";
import {
  Box,
  CardBox,
  Grid,
  HStack,
  LStack,
  WStack,
  styled,
} from "@/styled-system/jsx";

import { Props, useListingForm } from "./useListingForm";

const twoColumns = {
  base: "minmax(0, 1fr)",
  md: "repeat(2, minmax(0, 1fr))",
};

export function ListingFormScreen(props: Props) {
  const t = useTranslation();
  const categories = useMarketCategories();
  const phone = useAccountPhoneGet();
  const settings = useSettings();
  const {
    form,
    priceType,
    assets,
    setAssets,
    uploading,
    setUploading,
    publishing,
    onSubmit,
  } = useListingForm(props);
  const errors = form.formState.errors;

  if (!phone.data) {
    return <UnreadyBanner error={phone.error} />;
  }

  const title = props.listing
    ? t.marketplace.form.editTitle
    : t.marketplace.form.createTitle;

  const durationDays = settings.ready
    ? settings.settings.marketplace_listing_duration_days
    : undefined;

  const visibility = props.listing
    ? props.listing.is_expired
      ? undefined
      : t.marketplace.form.visibilityCurrent
          .replace("{date}", formatDate(props.listing.expires_at))
          .replace("{remaining}", formatRemaining(t, props.listing.expires_at))
    : durationDays
      ? t.marketplace.form.visibilityNotice
          .replace("{days}", String(durationDays))
          .replace("{date}", formatDate(addDays(durationDays)))
      : undefined;

  return (
    <LStack gap="4" maxW="breakpoint-md" mx="auto" w="full">
      <Breadcrumbs
        index={{ href: "/flohmarkt", label: t.nav.marketplace }}
        crumbs={[
          ...(props.listing
            ? [
                {
                  label: props.listing.title,
                  href: `/flohmarkt/${props.listing.slug}`,
                },
              ]
            : []),
          { label: title, href: "#" },
        ]}
      />

      {!phone.data.phone && !props.listing ? (
        <PhoneRequiredNotice />
      ) : (
        <styled.form
          w="full"
          display="flex"
          flexDir="column"
          gap="4"
          position="relative"
          aria-busy={publishing}
          onSubmit={blockSubmitWhileUploading(uploading, onSubmit)}
        >
          {publishing && (
            <Box
              position="absolute"
              inset="0"
              zIndex="overlay"
              display="flex"
              flexDirection="column"
              alignItems="center"
              justifyContent="center"
              gap="3"
              borderRadius="l3"
              bg="bg.default/80"
              backdropBlur="md"
              backdropFilter="auto"
              data-testid="listing-publishing"
              aria-live="polite"
            >
              <Spinner size="lg" />
              <styled.p fontWeight="medium">
                {props.listing
                  ? t.marketplace.form.saving
                  : t.marketplace.form.creating}
              </styled.p>
              {!props.listing && (
                <styled.p fontSize="sm" color="fg.muted">
                  {t.marketplace.form.publishingHint}
                </styled.p>
              )}
            </Box>
          )}
          <Heading size="xl">{title}</Heading>

          <Section title={t.marketplace.form.sectionPhotos}>
            <FormControl>
              <ListingImagesField
                value={assets}
                onChange={setAssets}
                onUploadingChange={setUploading}
              />
              <FormHelperText>
                {t.marketplace.form.imagesHelp.replace(
                  "{max}",
                  String(MAX_LISTING_IMAGES),
                )}
              </FormHelperText>
            </FormControl>
          </Section>

          <Section title={t.marketplace.form.sectionWhat}>
            <FormControl>
              <FormLabel>{t.marketplace.form.title}</FormLabel>
              <Input
                placeholder={t.marketplace.form.titlePlaceholder}
                {...form.register("title")}
              />
              <FormErrorText>{errors.title?.message}</FormErrorText>
            </FormControl>

            <Grid w="full" gap="4" gridTemplateColumns={twoColumns}>
              <FormControl>
                <FormLabel>{t.marketplace.form.category}</FormLabel>
                <NativeSelect
                  options={categories.options}
                  {...form.register("category")}
                />
              </FormControl>

              <FormControl>
                <FormLabel>{t.marketplace.form.condition}</FormLabel>
                <NativeSelect
                  options={conditionOptions(t)}
                  {...form.register("condition")}
                />
              </FormControl>
            </Grid>
          </Section>

          <Section title={t.marketplace.form.sectionPrice}>
            <Grid w="full" gap="4" gridTemplateColumns={twoColumns}>
              <FormControl>
                <FormLabel>{t.marketplace.form.priceType}</FormLabel>
                <NativeSelect
                  options={priceTypeOptions(t)}
                  {...form.register("price_type")}
                />
              </FormControl>

              {priceType !== ListingPriceType.free && (
                <FormControl>
                  <FormLabel>{t.marketplace.form.price}</FormLabel>
                  <HStack gap="0" alignItems="stretch" w="full">
                    <InputPrefix>€</InputPrefix>
                    <Input
                      type="number"
                      inputMode="decimal"
                      min={0}
                      step="0.01"
                      flex="1"
                      borderTopLeftRadius="none"
                      borderBottomLeftRadius="none"
                      {...form.register("price")}
                    />
                  </HStack>
                  <FormErrorText>{errors.price?.message}</FormErrorText>
                </FormControl>
              )}
            </Grid>
          </Section>

          <Section title={t.marketplace.form.sectionDetails}>
            <FormControl>
              <FormLabel>{t.marketplace.form.description}</FormLabel>
              <styled.textarea
                w="full"
                minH="40"
                p="2"
                borderWidth="thin"
                borderStyle="solid"
                borderColor="border.default"
                borderRadius="l2"
                bg="bg.default"
                _placeholder={{ color: "fg.muted", opacity: "full" }}
                placeholder={t.marketplace.form.descriptionPlaceholder}
                {...form.register("description")}
              />
              <FormErrorText>{errors.description?.message}</FormErrorText>
            </FormControl>
          </Section>

          <WStack
            justifyContent="space-between"
            alignItems="center"
            flexWrap={{ base: "wrap", sm: "nowrap" }}
          >
            <styled.p
              fontSize="sm"
              color="fg.muted"
              data-testid="listing-visibility"
            >
              {visibility}
            </styled.p>
            <Button
              type="submit"
              size="lg"
              flexShrink="0"
              w={{ base: "full", sm: "auto" }}
              loading={publishing || uploading}
              loadingText={uploading ? t.toasts.uploading : undefined}
              disabled={uploading || publishing}
            >
              {props.listing
                ? t.marketplace.form.submitEdit
                : t.marketplace.form.submitCreate}
            </Button>
          </WStack>
        </styled.form>
      )}
    </LStack>
  );
}

function Section({ title, children }: PropsWithChildren<{ title: string }>) {
  return (
    <CardBox w="full" display="flex" flexDir="column" gap="4">
      <Heading size="md">{title}</Heading>
      {children}
    </CardBox>
  );
}
