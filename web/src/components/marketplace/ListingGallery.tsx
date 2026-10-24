"use client";

import { ChevronLeft, ChevronRight, Maximize2 } from "lucide-react";
import { useEffect, useRef, useState } from "react";

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
  const trackRef = useRef<HTMLDivElement>(null);
  const current = assets[index] ?? assets[0];

  useEffect(() => {
    const track = trackRef.current;
    if (!track || track.clientWidth === 0) return;

    const visible = Math.round(track.scrollLeft / track.clientWidth);
    if (visible !== index) {
      track.scrollTo({ left: index * track.clientWidth, behavior: "smooth" });
    }
  }, [index]);

  function handleScroll() {
    const track = trackRef.current;
    if (!track || track.clientWidth === 0) return;

    const visible = Math.round(track.scrollLeft / track.clientWidth);
    if (visible !== index && assets[visible]) {
      setIndex(visible);
    }
  }

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
        position="relative"
        w="full"
        aspectRatio="[4/3]"
        bg="bg.muted"
        borderRadius="lg"
        overflow="hidden"
        className="group"
      >
        <HStack
          ref={trackRef}
          onScroll={handleScroll}
          gap="0"
          w="full"
          h="full"
          overflowX="auto"
          overflowY="hidden"
          scrollSnapType="x"
          scrollSnapStrictness="mandatory"
          overscrollBehaviorX="contain"
          scrollbar="hidden"
        >
          {assets.map((a, i) => (
            <styled.button
              key={a.id}
              type="button"
              onClick={() => {
                setIndex(i);
                setOpen(true);
              }}
              aria-label={t.marketplace.lightbox.open}
              title={t.marketplace.lightbox.open}
              flexShrink="0"
              w="full"
              h="full"
              scrollSnapAlign="center"
              scrollSnapStop="always"
              cursor="zoom-in"
            >
              <styled.img
                src={getAssetURL(a.path)}
                alt={`${title} ${i + 1}`}
                loading="eager"
                decoding="async"
                draggable={false}
                w="full"
                h="full"
                objectFit="contain"
              />
            </styled.button>
          ))}
        </HStack>
        {assets.length > 1 && (
          <styled.span
            position="absolute"
            left="2"
            bottom="2"
            px="2"
            py="0.5"
            borderRadius="full"
            bg="black.a8"
            color="white"
            fontSize="xs"
            fontWeight="medium"
            pointerEvents="none"
          >
            {index + 1} / {assets.length}
          </styled.span>
        )}
        {assets.length > 1 && (
          <>
            <NavButton
              left="2"
              onClick={() =>
                setIndex((i) => (i - 1 + assets.length) % assets.length)
              }
              aria-label={t.marketplace.lightbox.previous}
              title={t.marketplace.lightbox.previous}
            >
              <ChevronLeft size={20} />
            </NavButton>
            <NavButton
              right="2"
              onClick={() => setIndex((i) => (i + 1) % assets.length)}
              aria-label={t.marketplace.lightbox.next}
              title={t.marketplace.lightbox.next}
            >
              <ChevronRight size={20} />
            </NavButton>
          </>
        )}
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
          pointerEvents="none"
          opacity={{ base: "full", md: "7" }}
          _groupHover={{ opacity: "full" }}
        >
          <Maximize2 size={16} />
        </Box>
      </Box>
      {assets.length > 1 && (
        <HStack gap="2" w="full" overflowX="auto" overscrollBehaviorX="contain">
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

const NavButton = styled(
  "button",
  {
    base: {
      position: "absolute",
      top: "0",
      bottom: "0",
      my: "auto",
      display: { base: "none", md: "flex" },
      alignItems: "center",
      justifyContent: "center",
      w: "9",
      h: "9",
      borderRadius: "full",
      bg: "black.a8",
      color: "white",
      cursor: "pointer",
      opacity: "0",
      transition: "opacity",
      transitionDuration: "fast",
      _groupHover: { opacity: "full" },
      _focusVisible: { opacity: "full" },
      _hover: { bg: "black.a10" },
    },
  },
  { defaultProps: { type: "button" } },
);
