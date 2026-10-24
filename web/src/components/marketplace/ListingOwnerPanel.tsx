"use client";

import { useRouter } from "next/navigation";

import { handle } from "@/api/client";
import {
  listingDelete,
  listingRenew,
  listingUpdate,
  useListingOfferList,
} from "@/api/openapi-client/listings";
import { Listing, ListingStatus } from "@/api/openapi-schema";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { Button } from "@/components/ui/button";
import { FormControl } from "@/components/ui/form/FormControl";
import { FormHelperText } from "@/components/ui/form/FormHelperText";
import { FormLabel } from "@/components/ui/form/FormLabel";
import { Heading } from "@/components/ui/heading";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { formatCents, formatDate } from "@/lib/marketplace/marketplace";
import { CardBox, HStack, LStack, styled } from "@/styled-system/jsx";

import { ContactButtons } from "./ContactButtons";
import { NativeSelect } from "./NativeSelect";
import { statusOptions } from "./options";

const RENEW_WINDOW_MS = 7 * 24 * 60 * 60 * 1000;

type Props = {
  listing: Listing;
  onChange: () => Promise<unknown>;
};

export function ListingOwnerPanel({ listing, onChange }: Props) {
  const t = useTranslation();
  const router = useRouter();
  const offers = useListingOfferList(listing.slug, {
    swr: { enabled: listing.can_manage },
  });

  const canRenew =
    new Date(listing.expires_at).getTime() - Date.now() <= RENEW_WINDOW_MS;

  async function handleStatus(status: ListingStatus) {
    await handle(
      async () => {
        await listingUpdate(listing.slug, { status });
        await onChange();
      },
      {
        promiseToast: {
          loading: t.marketplace.form.saving,
          success: t.marketplace.owner.statusUpdated,
        },
      },
    );
  }

  async function handleRenew() {
    await handle(
      async () => {
        await listingRenew(listing.slug);
        await onChange();
      },
      {
        promiseToast: {
          loading: t.marketplace.form.saving,
          success: t.marketplace.owner.renewed,
        },
      },
    );
  }

  async function handleDelete() {
    if (!window.confirm(t.marketplace.owner.deleteConfirm)) {
      return;
    }
    await handle(
      async () => {
        await listingDelete(listing.slug);
        router.push(
          listing.is_own ? "/flohmarkt?scope=mine" : "/flohmarkt?scope=manage",
        );
      },
      {
        promiseToast: {
          loading: t.marketplace.form.saving,
          success: t.marketplace.owner.deleted,
        },
      },
    );
  }

  const sellerMessage = t.marketplace.contact.whatsappSellerMessage.replace(
    "{title}",
    listing.title,
  );

  return (
    <LStack gap="4" w="full">
      <CardBox w="full">
        <LStack gap="3">
          <Heading size="sm">
            {listing.is_own
              ? t.marketplace.owner.title
              : t.marketplace.owner.moderationTitle}
          </Heading>

          {listing.can_manage && (
            <FormControl>
              <FormLabel>{t.marketplace.owner.setStatus}</FormLabel>
              <NativeSelect
                value={listing.status}
                onChange={(e) => handleStatus(e.target.value as ListingStatus)}
                options={statusOptions(t)}
              />
            </FormControl>
          )}

          <HStack gap="2" flexWrap="wrap">
            {listing.can_manage && (
              <LinkButton
                href={`/flohmarkt/${listing.slug}/edit`}
                size="sm"
                variant="outline"
              >
                {t.marketplace.owner.edit}
              </LinkButton>
            )}
            {listing.can_manage && (
              <Button
                type="button"
                size="sm"
                variant="outline"
                disabled={!canRenew}
                onClick={handleRenew}
              >
                {t.marketplace.owner.renew}
              </Button>
            )}
            <Button
              type="button"
              size="sm"
              variant="ghost"
              colorPalette="red"
              onClick={handleDelete}
            >
              {t.marketplace.owner.delete}
            </Button>
          </HStack>
          {listing.can_manage && !canRenew && (
            <FormHelperText>{t.marketplace.owner.renewHelp}</FormHelperText>
          )}
        </LStack>
      </CardBox>

      {listing.can_manage && (
        <CardBox w="full" data-testid="listing-offers">
          <LStack gap="3">
            <Heading size="sm">{t.marketplace.owner.offersTitle}</Heading>
            {offers.data?.offers.length === 0 && (
              <styled.p fontSize="sm" color="fg.muted">
                {t.marketplace.owner.noOffers}
              </styled.p>
            )}
            {offers.data?.offers.map((o) => (
              <LStack
                key={o.id}
                gap="2"
                w="full"
                pb="3"
                borderBottomWidth="thin"
                borderBottomStyle="solid"
                borderColor="border.subtle"
              >
                <HStack justify="space-between" w="full">
                  <MemberBadge
                    profile={o.buyer}
                    size="sm"
                    name="full-horizontal"
                  />
                  <styled.span fontWeight="bold">
                    {formatCents(o.amount)}
                  </styled.span>
                </HStack>
                {o.message && (
                  <styled.p fontSize="sm" whiteSpace="pre-wrap">
                    {o.message}
                  </styled.p>
                )}
                <styled.span fontSize="xs" color="fg.muted">
                  {formatDate(o.updatedAt)}
                </styled.span>
                {o.buyer_phone && (
                  <ContactButtons
                    phone={o.buyer_phone}
                    message={sellerMessage}
                  />
                )}
              </LStack>
            ))}
          </LStack>
        </CardBox>
      )}
    </LStack>
  );
}
