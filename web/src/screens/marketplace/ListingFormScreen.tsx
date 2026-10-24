"use client";

import { useAccountPhoneGet } from "@/api/openapi-client/accounts";
import { ListingPriceType } from "@/api/openapi-schema";
import {
  ListingImagesField,
  MAX_LISTING_IMAGES,
} from "@/components/marketplace/ListingImagesField";
import { NativeSelect } from "@/components/marketplace/NativeSelect";
import { PhoneRequiredNotice } from "@/components/marketplace/PhoneRequiredNotice";
import { useMarketCategories } from "@/components/marketplace/useMarketCategories";
import {
  conditionOptions,
  priceTypeOptions,
} from "@/components/marketplace/options";
import { UnreadyBanner } from "@/components/site/Unready";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import { CardBox, Grid, LStack, WStack, styled } from "@/styled-system/jsx";
import { lstack } from "@/styled-system/patterns";

import { Props, useListingForm } from "./useListingForm";

export function ListingFormScreen(props: Props) {
  const t = useTranslation();
  const categories = useMarketCategories();
  const phone = useAccountPhoneGet();
  const {
    form,
    priceType,
    assets,
    setAssets,
    uploading,
    setUploading,
    onSubmit,
  } = useListingForm(props);
  const errors = form.formState.errors;

  if (!phone.data) {
    return <UnreadyBanner error={phone.error} />;
  }

  const title = props.listing
    ? t.marketplace.form.editTitle
    : t.marketplace.form.createTitle;

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
        <styled.form w="full" onSubmit={onSubmit}>
          <CardBox className={lstack()} gap="6">
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
              <FormLabel>{t.marketplace.form.location}</FormLabel>
              <Input
                placeholder={t.marketplace.form.locationPlaceholder}
                {...form.register("location")}
              />
              <FormErrorText>{errors.location?.message}</FormErrorText>
            </FormControl>

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
                placeholder={t.marketplace.form.descriptionPlaceholder}
                {...form.register("description")}
              />
              <FormErrorText>{errors.description?.message}</FormErrorText>
            </FormControl>

            <WStack justifyContent="flex-end">
              <Button
                type="submit"
                loading={form.formState.isSubmitting}
                disabled={uploading}
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
