"use client";

import { PropsWithChildren } from "react";

import { useAccountPhoneGet } from "@/api/openapi-client/accounts";
import { ListingPriceType } from "@/api/openapi-schema";
import {
  ListingImagesField,
  MAX_LISTING_IMAGES,
} from "@/components/marketplace/ListingImagesField";
import { NativeSelect } from "@/components/marketplace/NativeSelect";
import { PhoneRequiredNotice } from "@/components/marketplace/PhoneRequiredNotice";
import {
  categoryOptions,
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
import { Input, InputPrefix } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import {
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
          onSubmit={onSubmit}
        >
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
                  options={categoryOptions(t)}
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
          </Section>

          <WStack justifyContent="flex-end">
            <Button
              type="submit"
              size="lg"
              w={{ base: "full", sm: "auto" }}
              loading={form.formState.isSubmitting}
              disabled={uploading}
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
