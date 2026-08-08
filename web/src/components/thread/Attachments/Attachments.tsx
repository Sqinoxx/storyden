"use client";

import { Download } from "lucide-react";
import { Asset } from "@/api/openapi-schema";
import { downloadAsset, getAssetURL, getCleanFilename } from "@/utils/asset";
import { HStack, styled } from "@/styled-system/jsx";

export function Attachments({ assets }: { assets: Asset[] }) {
  const files = assets.filter((a) => !a.mime_type.startsWith("image/"));

  if (files.length === 0) {
    return null;
  }

  return (
    <HStack w="full" flexWrap="wrap" gap="2">
      {files.map((a) => {
        const displayName = getCleanFilename(a.filename);
        const url = getAssetURL(a.path);
        return (
          <styled.a
            key={a.id}
            href={url}
            download={displayName}
            target="_blank"
            rel="noreferrer"
            // The plain href is kept as a fallback for middle-click and "save
            // link as", but the click path goes through downloadAsset so the
            // request carries session credentials across origins.
            onClick={(e) => {
              if (e.metaKey || e.ctrlKey || e.shiftKey || e.button !== 0) return;
              e.preventDefault();
              void downloadAsset(url, displayName);
            }}
            display="inline-flex"
            alignItems="center"
            gap="2"
            px="3"
            py="2"
            borderRadius="md"
            borderWidth="thin"
            borderColor="border.default"
            bgColor="bg.subtle"
            color="fg.default"
            textDecoration="none"
            fontSize="sm"
            fontWeight="medium"
            maxWidth="xs"
            overflow="hidden"
            _hover={{ bgColor: "bg.muted" }}
          >
            <styled.span flexShrink="0" fontSize="sm">
              📄
            </styled.span>
            <styled.span
              overflow="hidden"
              textOverflow="ellipsis"
              whiteSpace="nowrap"
              flex="1"
              minWidth="0"
            >
              {displayName}
            </styled.span>
            <styled.span display="flex" alignItems="center" color="fg.muted" flexShrink="0" ml="1">
              <Download size={14} />
            </styled.span>
          </styled.a>
        );
      })}
    </HStack>
  );
}