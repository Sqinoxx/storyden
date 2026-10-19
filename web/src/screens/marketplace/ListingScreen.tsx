"use client";

import { useListingGet } from "@/api/openapi-client/listings";
import { Listing, Permission } from "@/api/openapi-schema";
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
import { ReportIcon } from "@/components/ui/icons/Report";
import { useTranslation } from "@/lib/i18n";
import { formatDate, formatListingPrice } from "@/lib/marketplace/marketplace";
import {
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

  const facts = [
    {
      label: t.marketplace.detail.category,
      value: t.marketplace.categories[listing.category],
    },
    {
      label: t.marketplace.detail.condition,
      value: t.marketplace.conditions[listing.condition],
    },
    ...(listing.location
      ? [{ label: t.marketplace.detail.location, value: listing.location }]
      : []),
  ];

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
        gap="6"
        gridTemplateColumns={{ base: "repeat(1, 1fr)", lg: "3fr 2fr" }}
        alignItems="start"
      >
        <LStack gap="4" minW="0">
          <ListingGallery assets={listing.assets} title={listing.title} />

          <LStack gap="2" w="full">
            <WStack alignItems="start">
              <Heading size="lg" wordBreak="break-word">
                {listing.title}
              </Heading>
              <ListingFavouriteButton
                listing={listing}
                onChange={() => mutate()}
              />
            </WStack>
            <HStack gap="2">
              <styled.span
                fontSize="2xl"
                fontWeight="bold"
                data-testid="listing-price"
              >
                {formatListingPrice(t, listing)}
              </styled.span>
              <ListingStatusBadge status={listing.status} />
            </HStack>
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

          <styled.dl
            display="grid"
            gridTemplateColumns="auto 1fr"
            columnGap="4"
            rowGap="1"
            fontSize="sm"
          >
            {facts.map((f) => (
              <styled.div key={f.label} display="contents">
                <styled.dt color="fg.muted">{f.label}</styled.dt>
                <styled.dd>{f.value}</styled.dd>
              </styled.div>
            ))}
          </styled.dl>

          {listing.description && (
            <styled.p whiteSpace="pre-wrap" wordBreak="break-word">
              {listing.description}
            </styled.p>
          )}
        </LStack>

        <LStack gap="4" w="full">
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
            <ListingOfferPanel listing={listing} onOffered={() => mutate()} />
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
