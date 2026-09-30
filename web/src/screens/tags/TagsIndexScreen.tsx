"use client";

import { useState } from "react";

import { useTagList } from "@/api/openapi-client/tags";
import { TagListResult } from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { TagBadgeList } from "@/components/tag/TagBadgeList";
import { Breadcrumbs } from "@/components/ui/Breadcrumbs";
import { Input } from "@/components/ui/input";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { LStack } from "@/styled-system/jsx";

type Props = {
  initialTagList: TagListResult;
};

export function TagsIndexScreen(props: Props) {
  const t = useTranslation();
  const [search, setSearch] = useState("");
  const { data, error } = useTagList(
    {},
    { swr: { fallbackData: props.initialTagList } },
  );
  if (!data) {
    return <Unready error={error} />;
  }

  const query = search.toLowerCase().trim();
  const tags = data.tags
    .filter((t) => t.item_count > 0)
    .filter((t) => t.name.toLowerCase().includes(query))
    .sort((a, b) => b.item_count - a.item_count);

  return (
    <LStack gap="4">
      <LStack gap="1">
        <Breadcrumbs
          index={{
            label: t.tags.indexTitle,
            href: "/tags",
          }}
          crumbs={[]}
        />

        <Text textStyle="sm" color="fg.muted">
          {t.tags.indexSubtitle}
        </Text>
      </LStack>

      <Input
        placeholder={t.tags.searchPlaceholder}
        value={search}
        onChange={(e) => setSearch(e.target.value)}
        width="full"
      />

      {tags.length > 0 ? (
        <TagBadgeList tags={tags} showItemCount />
      ) : (
        <Text textStyle="sm" color="fg.muted">
          {t.tags.emptyState}
        </Text>
      )}
    </LStack>
  );
}
