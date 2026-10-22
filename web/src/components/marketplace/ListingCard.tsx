"use client";

import Link from "next/link";

import { Listing } from "@/api/openapi-schema";
import { LocationIcon } from "@/components/ui/icons/Location";
import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { useTranslation } from "@/lib/i18n";
import {
  formatDate,
  formatListingPrice,
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
    <styled.article
      className="group"
      display="flex"
      flexDir="column"
      minW="0"
      borderWidth="thin"
      borderStyle="solid"
      borderColor="border.subtle"
      borderRadius="xl"
      overflow="hidden"
      bg="bg.default"
      position="relative"
      isolation="isolate"
      transitionProperty="common"
      transitionDuration="normal"
      opacity={listing.is_expired ? "7" : undefined}
      _hover={{ borderColor: "border.default", shadow: "md" }}
      _focusWithin={{ borderColor: "border.accent" }}
      data-testid="listing-card"
    >
      <Box
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        display="flex"
        alignItems="center"
        justifyContent="center"
        overflow="hidden"
      >
        {coverURL ? (
          <styled.img
            src={coverURL}
            alt=""
            loading="lazy"
            w="full"
            h="full"
            objectFit="cover"
            transitionProperty="[transform]"
            transitionDuration="slow"
            _groupHover={{ transform: "scale(1.04)" }}
          />
        ) : (
          <MarketplaceIcon w="10" h="10" color="fg.subtle" />
        )}
      </Box>

      <Box position="absolute" top="2" left="2" pointerEvents="none">
        <ListingStatusBadge status={listing.status} />
      </Box>

      <Box position="absolute" top="2" right="2" zIndex="docked">
        <ListingFavouriteButton listing={listing} onChange={onChange} />
      </Box>

      <LStack gap="1" p={{ base: "2.5", md: "3" }} w="full" flexGrow="1">
        <styled.span
          fontWeight="bold"
          fontSize={{ base: "md", md: "lg" }}
          lineHeight="tight"
        >
          {formatListingPrice(t, listing)}
        </styled.span>

        <Link href={listingURL(listing)} style={{ width: "100%" }}>
          <styled.span
            display="block"
            fontSize="sm"
            fontWeight="medium"
            lineClamp="2"
            wordBreak="break-word"
            _before={{
              content: '""',
              position: "absolute",
              inset: "0",
            }}
          >
            {listing.title}
          </styled.span>
        </Link>

        <HStack gap="1" flexWrap="wrap" mt="auto" pt="1">
          <styled.span
            fontSize="xs"
            px="1.5"
            py="0.5"
            borderRadius="md"
            bg="bg.muted"
            color="fg.muted"
          >
            {t.marketplace.conditions[listing.condition]}
          </styled.span>
        </HStack>

        <HStack gap="1" fontSize="xs" color="fg.muted" minW="0">
          {listing.location && (
            <>
              <LocationIcon w="3" h="3" flexShrink="0" />
              <styled.span truncate>{listing.location}</styled.span>
              <styled.span>·</styled.span>
            </>
          )}
          <styled.span flexShrink="0">
            {formatDate(listing.createdAt)}
          </styled.span>
        </HStack>

        {listing.is_own && listing.offer_count !== undefined && (
          <styled.span fontSize="xs" fontWeight="medium" color="fg.accent">
            {t.marketplace.offers.replace(
              "{count}",
              String(listing.offer_count),
            )}
          </styled.span>
        )}
      </LStack>
    </styled.article>
  );
}
