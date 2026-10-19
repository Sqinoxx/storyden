"use client";

import { useState } from "react";

import { Asset } from "@/api/openapi-schema";
import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

export function ListingGallery({
  assets,
  title,
}: {
  assets: Asset[];
  title: string;
}) {
  const [index, setIndex] = useState(0);
  const current = assets[index] ?? assets[0];

  if (!current) {
    return (
      <Box
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        borderRadius="lg"
        display="flex"
        alignItems="center"
        justifyContent="center"
      >
        <MarketplaceIcon w="16" h="16" color="fg.subtle" />
      </Box>
    );
  }

  return (
    <LStack gap="2" w="full">
      <Box
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        borderRadius="lg"
        overflow="hidden"
      >
        <styled.img
          src={getAssetURL(current.path)}
          alt={title}
          w="full"
          h="full"
          objectFit="contain"
        />
      </Box>
      {assets.length > 1 && (
        <HStack gap="2" w="full" overflowX="auto">
          {assets.map((a, i) => (
            <styled.button
              key={a.id}
              type="button"
              onClick={() => setIndex(i)}
              flexShrink="0"
              w="16"
              h="16"
              borderRadius="md"
              overflow="hidden"
              borderWidth="medium"
              borderStyle="solid"
              borderColor={i === index ? "border.accent" : "border.subtle"}
              cursor="pointer"
              aria-label={`${title} ${i + 1}`}
            >
              <styled.img
                src={getAssetURL(a.path)}
                alt=""
                w="full"
                h="full"
                objectFit="cover"
              />
            </styled.button>
          ))}
        </HStack>
      )}
    </LStack>
  );
}
