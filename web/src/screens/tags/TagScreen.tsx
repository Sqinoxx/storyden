"use client";

import { useTagGet } from "@/api/openapi-client/tags";
import { DatagraphItemKind, Tag, TagName } from "@/api/openapi-schema";
import { DatagraphItemCard } from "@/components/datagraph/DatagraphItemCard";
import { Unready } from "@/components/site/Unready";
import { TagBadge } from "@/components/tag/TagBadge";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Heading } from "@/components/ui/heading";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { HStack, LStack } from "@/styled-system/jsx";

type Props = {
  slug: TagName;
  initialTag: Tag;
};

export function TagScreen(props: Props) {
  const t = useTranslation();
  const { data, error } = useTagGet(props.slug, {
    swr: { fallbackData: props.initialTag },
  });
  if (!data) {
    return <Unready error={error} />;
  }

  const tag = data;
  const threads = tag.items.filter(
    (item) => item.kind === DatagraphItemKind.thread,
  );
  const pages = tag.items.filter(
    (item) => item.kind === DatagraphItemKind.node,
  );

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

      {threads.length === 0 && pages.length === 0 && (
        <Text textStyle="sm" color="fg.muted">
          {t.tags.noItems}
        </Text>
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
