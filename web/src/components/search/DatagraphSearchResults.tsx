import { DatagraphSearchResult } from "@/api/openapi-schema";
import { EmptyState } from "@/components/site/EmptyState";

import { styled } from "@/styled-system/jsx";

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

  return (
    <styled.ol width="full" display="flex" flexDirection="column" gap="4">
      {result.items.map((v) => (
        <DatagraphItemCard key={v.ref.id} item={v} query={query} />
      ))}
    </styled.ol>
  );
}
