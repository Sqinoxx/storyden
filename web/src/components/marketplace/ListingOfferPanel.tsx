"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { handle } from "@/api/client";
import { useAccountPhoneGet } from "@/api/openapi-client/accounts";
import {
  listingOfferSubmit,
  useListingContactGet,
} from "@/api/openapi-client/listings";
import { Listing, ListingStatus } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormErrorText } from "@/components/ui/form/FormErrorText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { Input } from "@/components/ui/input";
import { useTranslation } from "@/lib/i18n";
import {
  centsToEuros,
  eurosToCents,
  formatCents,
} from "@/lib/marketplace/marketplace";
import { CardBox, LStack, styled } from "@/styled-system/jsx";

import { ContactButtons } from "./ContactButtons";
import { PhoneRequiredNotice } from "./PhoneRequiredNotice";

const FormSchema = z.object({
  amount: z.coerce.number().min(0).max(1000000),
  message: z.string().max(1000),
});
type Form = z.infer<typeof FormSchema>;

type Props = {
  listing: Listing;
  onOffered: () => Promise<unknown>;
};

export function ListingOfferPanel({ listing, onOffered }: Props) {
  const t = useTranslation();
  const phone = useAccountPhoneGet();
  const contact = useListingContactGet(listing.slug, {
    swr: { enabled: listing.has_offered },
  });

  const form = useForm<Form>({
    resolver: zodResolver(FormSchema),
    defaultValues: { amount: centsToEuros(listing.price), message: "" },
  });

  const unavailable =
    listing.is_expired || listing.status === ListingStatus.sold;

  const onSubmit = form.handleSubmit(async (data) => {
    const offer = await handle(
      async () =>
        listingOfferSubmit(listing.slug, {
          amount: eurosToCents(data.amount),
          message: data.message || undefined,
        }),
      {
        promiseToast: {
          loading: t.marketplace.offer.sending,
          success: t.marketplace.offer.sent,
        },
      },
    );
    if (offer) {
      await onOffered();
      await contact.mutate();
    }
  });

  const lastAmount = form.watch("amount");
  const whatsappMessage = t.marketplace.contact.whatsappMessage
    .replace("{title}", listing.title)
    .replace("{amount}", formatCents(eurosToCents(Number(lastAmount) || 0)));

  return (
    <LStack gap="4" w="full">
      {listing.has_offered && contact.data && (
        <CardBox w="full" data-testid="listing-contact">
          <LStack gap="2">
            <Heading size="sm">{t.marketplace.contact.title}</Heading>
            <ContactButtons
              phone={contact.data.phone}
              message={whatsappMessage}
            />
          </LStack>
        </CardBox>
      )}

      {unavailable ? (
        <styled.p color="fg.muted">{t.marketplace.offer.unavailable}</styled.p>
      ) : phone.data && !phone.data.phone ? (
        <PhoneRequiredNotice />
      ) : (
        <CardBox w="full">
          <styled.form
            display="flex"
            flexDir="column"
            gap="3"
            onSubmit={onSubmit}
          >
            <Heading size="sm">{t.marketplace.offer.title}</Heading>
            <styled.p fontSize="sm" color="fg.muted">
              {t.marketplace.offer.description}
            </styled.p>
            <FormControl>
              <FormLabel>{t.marketplace.offer.amount}</FormLabel>
              <Input
                type="number"
                inputMode="decimal"
                min={0}
                step="0.01"
                {...form.register("amount")}
              />
              <FormErrorText>
                {form.formState.errors.amount?.message}
              </FormErrorText>
            </FormControl>
            <FormControl>
              <FormLabel>{t.marketplace.offer.message}</FormLabel>
              <Input
                placeholder={t.marketplace.offer.messagePlaceholder}
                {...form.register("message")}
              />
            </FormControl>
            <Button type="submit" loading={form.formState.isSubmitting}>
              {listing.has_offered
                ? t.marketplace.offer.update
                : t.marketplace.offer.submit}
            </Button>
          </styled.form>
        </CardBox>
      )}
    </LStack>
  );
}
