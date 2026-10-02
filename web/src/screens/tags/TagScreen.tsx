"use client";

import { useMemo } from "react";

import { useTagGet } from "@/api/openapi-client/tags";
import { Tag, TagName } from "@/api/openapi-schema";
import { DatagraphItemCard } from "@/components/datagraph/DatagraphItemCard";
import { Unready } from "@/components/site/Unready";
import { TagBadge } from "@/components/tag/TagBadge";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, LStack } from "@/styled-system/jsx";

import { TagItemFilters } from "./TagItemFilters";
import {
  filterTagItems,
  getTagItemFacets,
  hasActiveTagItemFilters,
  isTagItem,
} from "./filterTagItems";
import { useTagScreenFilters } from "./useTagScreenFilters";

type Props = {
  slug: TagName;
  initialTag: Tag;
};

export function TagScreen(props: Props) {
  const t = useTranslation();
  const { filters, set, reset } = useTagScreenFilters();
  const { data, error } = useTagGet(props.slug, {
    swr: { fallbackData: props.initialTag },
  });

  const items = data?.items;
  const facets = useMemo(() => getTagItemFacets(items ?? []), [items]);
  const filtered = useMemo(
    () => filterTagItems(items ?? [], filters),
    [items, filters],
  );

  if (!data) {
    return <Unready error={error} />;
  }

  const tag = data;
  const total = tag.items.filter(isTagItem).length;
  const filtering = hasActiveTagItemFilters(filters);
  const threads = filtered.filter((item) => item.kind === "thread");
  const pages = filtered.filter((item) => item.kind === "node");

  return (
    <LStack gap="6">
      <LStack gap="1">
        <Breadcrumbs
          index={{
            label: t.tags.indexTitle,
            href: "/tags",
          }}
          crumbs={[
            {
              label: tag.name,
              href: `/tags/${encodeURIComponent(tag.name)}`,
            },
          ]}
        />

        <HStack gap="1" flexWrap="wrap">
          <Text textStyle="sm" color="fg.muted">
            {t.tags.taggedWith}
          </Text>
          <TagBadge tag={tag} />
        </HStack>
      </LStack>

      {total === 0 ? (
        <Text textStyle="sm" color="fg.muted">
          {t.tags.noItems}
        </Text>
      ) : (
        <>
          <TagItemFilters
            filters={filters}
            onChange={set}
            categories={facets.categories}
            authors={facets.authors}
          />

          {filtering && (
            <HStack gap="3" flexWrap="wrap">
              <Text textStyle="xs" color="fg.muted" fontWeight="medium">
                {t.tags.resultCount
                  .replace("{shown}", String(filtered.length))
                  .replace("{total}", String(total))}
              </Text>
              <Button size="xs" variant="ghost" onClick={reset}>
                {t.tags.resetFilters}
              </Button>
            </HStack>
          )}

          {filtered.length === 0 && (
            <Box
              w="full"
              py="12"
              textAlign="center"
              borderRadius="l3"
              borderWidth="thin"
              borderStyle="dashed"
              borderColor="border.subtle"
            >
              <Text textStyle="sm" color="fg.muted">
                {t.tags.noFilterResults}
              </Text>
            </Box>
          )}
        </>
      )}

      {threads.length > 0 && (
        <LStack gap="3">
          <Heading size="md">{t.tags.threadsHeading}</Heading>
          {threads.map((item) => (
            <DatagraphItemCard key={item.ref.id} item={item} />
          ))}
        </LStack>
      )}

      {pages.length > 0 && (
        <LStack gap="3">
          <Heading size="md">{t.tags.pagesHeading}</Heading>
          {pages.map((item) => (
            <DatagraphItemCard key={item.ref.id} item={item} />
          ))}
        </LStack>
      )}
    </LStack>
  );
}
