"use client";

import { useRef, useState } from "react";

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
  const track = useRef<HTMLDivElement>(null);

  if (assets.length === 0) {
    return (
      <Box
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        borderRadius="xl"
        display="flex"
        alignItems="center"
        justifyContent="center"
      >
        <MarketplaceIcon w="16" h="16" color="fg.subtle" />
      </Box>
    );
  }

  function handleScroll() {
    const el = track.current;
    if (!el) return;
    setIndex(Math.round(el.scrollLeft / el.clientWidth));
  }

  function goTo(i: number) {
    const el = track.current;
    if (!el) return;
    el.scrollTo({ left: i * el.clientWidth, behavior: "smooth" });
    setIndex(i);
  }

  return (
    <LStack gap="2" w="full">
      <Box
        position="relative"
        w="full"
        borderRadius="xl"
        overflow="hidden"
        bg="bg.muted"
      >
        <styled.div
          ref={track}
          display="flex"
          w="full"
          aspectRatio="[4/3]"
          overflowX="auto"
          scrollSnapType="x"
          scrollSnapStrictness="mandatory"
          style={{ scrollbarWidth: "none" }}
          onScroll={handleScroll}
        >
          {assets.map((a, i) => (
            <styled.img
              key={a.id}
              src={getAssetURL(a.path)}
              alt={i === 0 ? title : `${title} ${i + 1}`}
              flexShrink="0"
              w="full"
              h="full"
              objectFit="contain"
              scrollSnapAlign="start"
              scrollSnapStop="always"
            />
          ))}
        </styled.div>

        {assets.length > 1 && (
          <styled.span
            position="absolute"
            bottom="2"
            right="2"
            px="2"
            py="0.5"
            borderRadius="full"
            bg="bg.default"
            fontSize="xs"
            fontWeight="medium"
            shadow="xs"
            aria-hidden
          >
            {index + 1} / {assets.length}
          </styled.span>
        )}
      </Box>

      {assets.length > 1 && (
        <HStack gap="2" w="full" overflowX="auto" hideBelow="md">
          {assets.map((a, i) => (
            <styled.button
              key={a.id}
              type="button"
              onClick={() => goTo(i)}
              flexShrink="0"
              w="16"
              h="16"
              borderRadius="md"
              overflow="hidden"
              borderWidth="medium"
              borderStyle="solid"
              borderColor={i === index ? "border.accent" : "transparent"}
              opacity={i === index ? undefined : "7"}
              cursor="pointer"
              _hover={{ opacity: "full" }}
              aria-label={`${title} ${i + 1}`}
              aria-current={i === index}
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
