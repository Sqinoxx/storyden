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
          display="flex"
          alignItems="center"
          gap="1"
          px="2"
          py="1"
          borderRadius="md"
          bgColor="bg.subtle"
          fontSize="sm"
          _hover={{ bgColor: "bg.muted" }}
        >
          📄 {a.filename}
        </styled.a>
      ))}
    </HStack>
  );
}