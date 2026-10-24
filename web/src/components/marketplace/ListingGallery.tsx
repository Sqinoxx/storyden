"use client";

import { Maximize2 } from "lucide-react";
import { useState } from "react";

import { Asset } from "@/api/openapi-schema";
import { MarketplaceIcon } from "@/components/ui/icons/Marketplace";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, LStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

import { ListingLightbox } from "./ListingLightbox";

export function ListingGallery({
  assets,
  title,
}: {
  assets: Asset[];
  title: string;
}) {
  const t = useTranslation();
  const [index, setIndex] = useState(0);
  const [open, setOpen] = useState(false);
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
      <styled.button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={t.marketplace.lightbox.open}
        title={t.marketplace.lightbox.open}
        position="relative"
        display="block"
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        borderRadius="lg"
        overflow="hidden"
        cursor="zoom-in"
        className="group"
      >
        <styled.img
          src={getAssetURL(current.path)}
          alt={title}
          w="full"
          h="full"
          objectFit="contain"
        />
        <Box
          position="absolute"
          right="2"
          bottom="2"
          display="flex"
          alignItems="center"
          justifyContent="center"
          w="8"
          h="8"
          borderRadius="full"
          bg="black.a8"
          color="white"
          opacity={{ base: "full", md: "7" }}
          _groupHover={{ opacity: "full" }}
        >
          <Maximize2 size={16} />
        </Box>
      </styled.button>
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
      {open && (
        <ListingLightbox
          assets={assets}
          index={index}
          title={title}
          onIndexChange={setIndex}
          onClose={() => setOpen(false)}
        />
      )}
    </LStack>
  );
}
