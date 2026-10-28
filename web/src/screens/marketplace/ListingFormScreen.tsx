"use client";

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
import { Input } from "@/components/ui/input";
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
  LStack,
  WStack,
  styled,
} from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useListingForm } from "./useListingForm";

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
    <LStack gap="4">
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
          position="relative"
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
          <CardBox className={lstack()} gap="6" aria-busy={publishing}>
            <Heading size="md">{title}</Heading>

            <FormControl>
              <FormLabel>{t.marketplace.form.images}</FormLabel>
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

            <FormControl>
              <FormLabel>{t.marketplace.form.title}</FormLabel>
              <Input
                placeholder={t.marketplace.form.titlePlaceholder}
                {...form.register("title")}
              />
              <FormErrorText>{errors.title?.message}</FormErrorText>
            </FormControl>

            <Grid
              w="full"
              gap="4"
              gridTemplateColumns={{
                base: "repeat(1, 1fr)",
                md: "repeat(2, 1fr)",
              }}
            >
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
                  <Input
                    type="number"
                    inputMode="decimal"
                    min={0}
                    step="0.01"
                    {...form.register("price")}
                  />
                  <FormErrorText>{errors.price?.message}</FormErrorText>
                </FormControl>
              )}

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

            <WStack
              justifyContent="space-between"
              alignItems="center"
              flexWrap={{ base: "wrap", md: "nowrap" }}
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
                flexShrink="0"
                loading={publishing || uploading}
                loadingText={uploading ? t.toasts.uploading : undefined}
                disabled={uploading || publishing}
              >
                {props.listing
                  ? t.marketplace.form.submitEdit
                  : t.marketplace.form.submitCreate}
              </Button>
            </WStack>
          </CardBox>
        </styled.form>
      )}
    </LStack>
  );
}
