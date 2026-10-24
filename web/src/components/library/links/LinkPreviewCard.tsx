"use client";

import { useState } from "react";

import { LinkReference } from "@/api/openapi-schema";
import { ExternalLinkIcon } from "@/components/ui/icons/ExternalLink";
import { LinkIcon } from "@/components/ui/icons/Link";
import { HStack, VStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";

type Props = {
  link: LinkReference;
  disableAnchors?: boolean;
};

export function LinkPreviewCard({ link, disableAnchors }: Props) {
  const [imageFailed, setImageFailed] = useState(false);
  const [faviconFailed, setFaviconFailed] = useState(false);

  const image = getAssetURL(link.primary_image?.path);
  const favicon = getAssetURL(link.favicon_image?.path);
  const title = link.title?.trim() || prettyURL(link.url);
  const description = link.description?.trim();

  const showImage = Boolean(image) && !imageFailed;
  const showFavicon = Boolean(favicon) && !faviconFailed;

  return (
    <styled.a
      href={link.url}
      target="_blank"
      rel="noopener noreferrer"
      onClick={(e) => disableAnchors && e.preventDefault()}
      display="flex"
      w="full"
      maxW="prose"
      minH="20"
      overflow="hidden"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.default"
      borderLeftWidth="medium"
      borderLeftColor="fg.accent"
      bg="bg.subtle"
      textDecoration="none"
      color="fg.default"
      transitionProperty="colors"
      transitionDuration="normal"
      _hover={{ bg: "bg.muted", textDecoration: "none" }}
      _focusVisible={{
        outlineWidth: "medium",
        outlineStyle: "solid",
        outlineColor: "border.accent",
      }}
    >
      <VStack flex="1" minW="0" alignItems="start" gap="1" p="3">
        <HStack gap="1.5" minW="0" w="full" color="fg.muted" fontSize="xs">
          {showFavicon ? (
            <styled.img
              src={favicon}
              alt=""
              w="4"
              h="4"
              flexShrink="0"
              borderRadius="xs"
              onError={() => setFaviconFailed(true)}
            />
          ) : (
            <LinkIcon w="3.5" h="3.5" flexShrink="0" />
          )}
          <styled.span truncate>{link.domain}</styled.span>
          <ExternalLinkIcon w="3" h="3" flexShrink="0" ml="auto" />
        </HStack>

        <styled.span
          fontWeight="semibold"
          lineHeight="tight"
          lineClamp="2"
          wordBreak="break-word"
        >
          {title}
        </styled.span>

        {description && (
          <styled.span
            fontSize="sm"
            color="fg.muted"
            lineHeight="normal"
            lineClamp="2"
          >
            {description}
          </styled.span>
        )}
      </VStack>

      {showImage ? (
        <Thumbnail>
          <styled.img
            src={image}
            alt=""
            position="absolute"
            inset="0"
            w="full"
            h="full"
            objectFit="cover"
            onError={() => setImageFailed(true)}
          />
        </Thumbnail>
      ) : (
        showFavicon && (
          <Thumbnail display="flex" alignItems="center" justifyContent="center">
            <styled.img
              src={favicon}
              alt=""
              w={{ base: "10", sm: "12" }}
              h={{ base: "10", sm: "12" }}
              objectFit="contain"
              borderRadius="md"
            />
          </Thumbnail>
        )
      )}
    </styled.a>
  );
}

const Thumbnail = styled("div", {
  base: {
    flexShrink: "0",
    position: "relative",
    width: { base: "24", sm: "32" },
    bg: "bg.muted",
    borderLeftWidth: "thin",
    borderLeftColor: "border.subtle",
    overflow: "hidden",
  },
});

function prettyURL(url: string) {
  return url.replace(/^https?:\/\/(www\.)?/, "").replace(/\/$/, "");
}
