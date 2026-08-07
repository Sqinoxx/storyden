import { FileText } from "lucide-react";
import { Fragment } from "react";

import { DatagraphAssetMatch } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack, styled } from "@/styled-system/jsx";
import { getCleanFilename } from "@/utils/asset";

type Props = {
  matches?: DatagraphAssetMatch[];
  query?: string;
};

export function AssetMatchList({ matches, query }: Props) {
  if (!matches || matches.length === 0) return null;

  return (
    <LStack
      gap="2"
      w="full"
      borderTopWidth="thin"
      borderColor="border.subtle"
      pt="2"
      mt="1"
    >
      {matches.map((match, i) => (
        <AssetMatchRow key={`${match.asset.id}-${i}`} match={match} query={query} />
      ))}
    </LStack>
  );
}

function AssetMatchRow({
  match,
  query,
}: {
  match: DatagraphAssetMatch;
  query?: string;
}) {
  const t = useTranslation();
  const filename = getCleanFilename(match.asset.filename);

  return (
    <LStack gap="1" w="full" minWidth="0">
      <HStack gap="1" color="fg.muted" fontSize="xs">
        <FileText size={14} />
        <styled.span fontWeight="medium">{t.ocr.matchInAttachment}</styled.span>
        <styled.span color="fg.subtle">·</styled.span>
        <styled.span
          overflow="hidden"
          whiteSpace="nowrap"
          textOverflow="ellipsis"
          minWidth="0"
        >
          {filename}
        </styled.span>
      </HStack>

      <styled.p
        fontSize="sm"
        color="fg.subtle"
        bg="bg.subtle"
        borderWidth="thin"
        borderColor="border.subtle"
        borderRadius="lg"
        px="3"
        py="2"
        w="full"
      >
        <HighlightedExcerpt text={match.excerpt} query={query} />
      </styled.p>
    </LStack>
  );
}

function HighlightedExcerpt({
  text,
  query,
}: {
  text: string;
  query?: string;
}) {
  const term = query?.trim();
  if (!term) return <>{text}</>;

  const escaped = term.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
  const parts = text.split(new RegExp(`(${escaped})`, "gi"));

  return (
    <>
      {parts.map((part, i) =>
        part.toLowerCase() === term.toLowerCase() ? (
          <styled.mark
            key={i}
            bg="amber.a4"
            color="amber.11"
            borderRadius="sm"
            px="0.5"
          >
            {part}
          </styled.mark>
        ) : (
          <Fragment key={i}>{part}</Fragment>
        ),
      )}
    </>
  );
}
