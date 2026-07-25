import { Download } from "lucide-react";
import { Asset } from "@/api/openapi-schema";
import { getAssetURL } from "@/utils/asset";
import { HStack, styled } from "@/styled-system/jsx";

export function Attachments({ assets }: { assets: Asset[] }) {
  const files = assets.filter((a) => !a.mime_type.startsWith("image/"));

  if (files.length === 0) {
    return null;
  }

  return (
    <HStack w="full" flexWrap="wrap" gap="2">
      {files.map((a) => (
        <styled.a
          key={a.id}
          href={getAssetURL(a.path)}
          download={a.filename}
          target="_blank"
          rel="noreferrer"
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
            {a.filename}
          </styled.span>
          <styled.span display="flex" alignItems="center" color="fg.muted" flexShrink="0" ml="1">
            <Download size={14} />
          </styled.span>
        </styled.a>
      ))}
    </HStack>
  );
}