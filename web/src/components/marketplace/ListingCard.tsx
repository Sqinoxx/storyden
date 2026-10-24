"use client";

import Link from "next/link";

import { Listing } from "@/api/openapi-schema";
import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { useTranslation } from "@/lib/i18n";
import {
  formatDate,
  formatListingPrice,
  formatRemaining,
  listingURL,
} from "@/lib/marketplace/marketplace";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

import { ListingFavouriteButton } from "./ListingFavouriteButton";
import { ListingStatusBadge } from "./ListingStatusBadge";

type Props = {
  listing: Listing;
  onChange?: () => void;
};

export function ListingCard({ listing, onChange }: Props) {
  const t = useTranslation();
  const cover = listing.assets[0];
  const coverURL = cover ? getAssetURL(cover.path) : undefined;

  return (
    <LStack
      gap="0"
      borderWidth="thin"
      borderStyle="solid"
      borderColor="border.subtle"
      borderRadius="lg"
      overflow="hidden"
      bg="bg.default"
      position="relative"
      opacity={listing.is_expired ? "7" : undefined}
      data-testid="listing-card"
    >
      <Link href={listingURL(listing)} style={{ width: "100%" }}>
        <Box
          w="full"
          aspectRatio="1"
          bg="bg.muted"
          display="flex"
          alignItems="center"
          justifyContent="center"
          overflow="hidden"
        >
          {coverURL ? (
            <styled.img
              src={coverURL}
              alt={listing.title}
              w="full"
              h="full"
              objectFit="cover"
            />
          ) : (
            <MarketplaceIcon w="12" h="12" color="fg.subtle" />
          )}
        </Box>
      </Link>

      <Box position="absolute" top="2" right="2">
        <ListingFavouriteButton listing={listing} onChange={onChange} />
      </Box>

      <LStack gap="1" p="3" w="full">
        <HStack w="full" justify="space-between" gap="2">
          <styled.span fontWeight="bold" fontSize="lg">
            {formatListingPrice(t, listing)}
          </styled.span>
          <ListingStatusBadge status={listing.status} />
        </HStack>
        <Link href={listingURL(listing)} style={{ width: "100%" }}>
          <styled.span
            display="block"
            fontWeight="medium"
            lineClamp="2"
            wordBreak="break-word"
          >
            {listing.title}
          </styled.span>
        </Link>
        <styled.span fontSize="xs" color="fg.muted">
          {formatDate(listing.createdAt)}
        </styled.span>
        {listing.is_own && !listing.is_expired && (
          <styled.span
            fontSize="xs"
            color="fg.muted"
            data-testid="listing-card-remaining"
          >
            {formatRemaining(t, listing.expires_at)}
          </styled.span>
        )}
        {listing.is_own && listing.offer_count !== undefined && (
          <styled.span fontSize="xs" color="fg.muted">
            {t.marketplace.offers.replace(
              "{count}",
              String(listing.offer_count),
            )}
          </styled.span>
        )}
      </LStack>
    </LStack>
  );
}
