"use client";

import { ReactNode } from "react";

import { useListingGet } from "@/api/openapi-client/listings";
import { Listing, ListingStatus, Permission } from "@/api/openapi-schema";
import { useSession } from "@/auth";
import { ListingFavouriteButton } from "@/components/marketplace/ListingFavouriteButton";
import { ListingGallery } from "@/components/marketplace/ListingGallery";
import { ListingOfferPanel } from "@/components/marketplace/ListingOfferPanel";
import { ListingOwnerPanel } from "@/components/marketplace/ListingOwnerPanel";
import { ListingStatusBadge } from "@/components/marketplace/ListingStatusBadge";
import { MemberBadge } from "@/components/member/MemberBadge/MemberBadge";
import { ReportListingModal } from "@/components/report/ReportListingModal";
import { UnreadyBanner } from "@/components/site/Unready";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { LocationIcon } from "@/components/ui/icons/Location";
import { ReportIcon } from "@/components/ui/icons/Report";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { formatDate, formatListingPrice } from "@/lib/marketplace/marketplace";
import {
  Box,
  CardBox,
  Grid,
  HStack,
  LStack,
  WStack,
  styled,
} from "@/styled-system/jsx";
import { hasPermission } from "@/utils/permissions";
import { useDisclosure } from "@/utils/useDisclosure";

type Props = {
  slug: string;
  initialListing?: Listing;
};

export function ListingScreen({ slug, initialListing }: Props) {
  const t = useTranslation();
  const session = useSession();
  const report = useDisclosure();
  const { data, error, mutate } = useListingGet(slug, {
    swr: { fallbackData: initialListing },
  });

  if (!data) {
    return <UnreadyBanner error={error} />;
  }

  const listing = data;
  const isModerator = hasPermission(session, Permission.MANAGE_LISTINGS);
  const available =
    !listing.is_expired && listing.status !== ListingStatus.sold;

  return (
    <LStack gap="4">
      <Breadcrumbs
        index={{ href: "/flohmarkt", label: t.nav.marketplace }}
        crumbs={[{ label: listing.title, href: `/flohmarkt/${listing.slug}` }]}
      />

      {listing.is_expired && (
        <CardBox w="full" bg="bg.warning" data-testid="listing-expired">
          <styled.p fontSize="sm">{t.marketplace.detail.expired}</styled.p>
        </CardBox>
      )}

      <Grid
        w="full"
        gap={{ base: "5", lg: "8" }}
        gridTemplateColumns={{
          base: "minmax(0, 1fr)",
          lg: "minmax(0, 3fr) minmax(0, 2fr)",
        }}
        alignItems="start"
      >
        <LStack gap="5" minW="0">
          <ListingGallery assets={listing.assets} title={listing.title} />

          <LStack gap="3" w="full">
            <WStack alignItems="start" gap="3">
              <Heading size="xl" wordBreak="break-word">
                {listing.title}
              </Heading>
              <ListingFavouriteButton
                listing={listing}
                onChange={() => mutate()}
              />
            </WStack>

            <HStack gap="3" flexWrap="wrap">
              <styled.span
                fontSize="3xl"
                fontWeight="bold"
                lineHeight="tight"
                data-testid="listing-price"
              >
                {formatListingPrice(t, listing)}
              </styled.span>
              <ListingStatusBadge status={listing.status} />
            </HStack>

            <HStack gap="2" flexWrap="wrap">
              <Fact label={t.marketplace.detail.category}>
                {t.marketplace.categories[listing.category]}
              </Fact>
              <Fact label={t.marketplace.detail.condition}>
                {t.marketplace.conditions[listing.condition]}
              </Fact>
              {listing.location && (
                <Fact label={t.marketplace.detail.location}>
                  <LocationIcon w="3.5" h="3.5" />
                  {listing.location}
                </Fact>
              )}
            </HStack>

            {!listing.is_own && available && (
              <LinkButton href="#offer" w="full" hideFrom="lg">
                {t.marketplace.offer.title}
              </LinkButton>
            )}
          </LStack>

          {listing.description && (
            <LStack
              gap="2"
              w="full"
              pt="5"
              borderTopWidth="thin"
              borderTopStyle="solid"
              borderColor="border.subtle"
            >
              <Heading size="md">{t.marketplace.detail.description}</Heading>
              <styled.p
                whiteSpace="pre-wrap"
                wordBreak="break-word"
                lineHeight="relaxed"
              >
                {listing.description}
              </styled.p>
            </LStack>
          )}

          <styled.span fontSize="xs" color="fg.muted">
            {t.marketplace.postedOn.replace(
              "{date}",
              formatDate(listing.createdAt),
            )}
            {" · "}
            {(listing.is_expired
              ? t.marketplace.expiredOn
              : t.marketplace.expiresOn
            ).replace("{date}", formatDate(listing.expires_at))}
          </styled.span>
        </LStack>

        <LStack gap="4" w="full" position={{ lg: "sticky" }} top={{ lg: "4" }}>
          <CardBox w="full">
            <LStack gap="2">
              <styled.span fontSize="xs" color="fg.muted">
                {t.marketplace.detail.seller}
              </styled.span>
              <MemberBadge
                profile={listing.owner}
                size="md"
                name="full-horizontal"
              />
            </LStack>
          </CardBox>

          {!listing.is_own && (
            <Box id="offer" w="full" scrollMarginTop="4">
              <ListingOfferPanel listing={listing} onOffered={() => mutate()} />
            </Box>
          )}

          {(listing.is_own || isModerator) && (
            <ListingOwnerPanel listing={listing} onChange={() => mutate()} />
          )}

          {!listing.is_own && (
            <Button
              type="button"
              size="xs"
              variant="ghost"
              alignSelf="start"
              color="fg.muted"
              onClick={report.onOpen}
            >
              <ReportIcon />
              {t.marketplace.detail.report}
            </Button>
          )}
        </LStack>
      </Grid>

      <ReportListingModal listing={listing} {...report} />
    </LStack>
  );
}

function Fact({ label, children }: { label: string; children: ReactNode }) {
  return (
    <HStack
      gap="1"
      px="2.5"
      py="1"
      fontSize="sm"
      borderRadius="full"
      bg="bg.muted"
      title={label}
    >
      <styled.span srOnly>{label}:</styled.span>
      {children}
    </HStack>
  );
}
