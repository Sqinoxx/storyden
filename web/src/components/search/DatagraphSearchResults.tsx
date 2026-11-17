import { DatagraphSearchResult } from "@/api/openapi-schema";
import { EmptyState } from "@/components/site/EmptyState";

import { IntelligenceIcon } from "@/components/ui/icons/Intelligence";
import { HStack, LStack, styled } from "@/styled-system/jsx";

import { DatagraphItemCard } from "../datagraph/DatagraphItemCard";
import { useTranslation } from "@/lib/i18n";

type Props = {
  result: DatagraphSearchResult;
  query?: string;
};

export function DatagraphSearchResults({ result, query }: Props) {
  const t = useTranslation();
  if (!result.items?.length) {
    return (
      <EmptyState>
        <p>{t.search.noItems}</p>
      </EmptyState>
    );
  }

  const semantic = new Set(result.semantic_matches ?? []);

  return (
    <styled.ol width="full" display="flex" flexDirection="column" gap="4">
      {result.items.map((v) =>
        semantic.has(v.ref.id) ? (
          <LStack key={v.ref.id} gap="1" data-testid="semantic-match">
            <HStack
              gap="1"
              color="fg.muted"
              fontSize="xs"
              title={t.search.semanticMatchHint}
            >
              <IntelligenceIcon width="3" height="3" />
              {t.search.semanticMatch}
            </HStack>
            <DatagraphItemCard item={v} query={query} />
          </LStack>
        ) : (
          <DatagraphItemCard key={v.ref.id} item={v} query={query} />
        ),
      )}
    </styled.ol>
  );
}
