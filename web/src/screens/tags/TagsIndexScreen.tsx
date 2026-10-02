"use client";

import Link from "next/link";
import { useState } from "react";

import { useTagList } from "@/api/openapi-client/tags";
import { TagListResult, TagReference } from "@/api/openapi-schema";
import { Unready } from "@/components/site/Unready";
import { badgeColourCSS } from "@/components/tag/TagBadge";
import { Button } from "@/components/ui/button";
import { Heading } from "@/components/ui/heading";
import { SearchIcon } from "@/components/ui/icons/Search";
import { TagIcon } from "@/components/ui/icons/Tag";
import { Input } from "@/components/ui/input";
import { InputGroup } from "@/components/ui/input-group";
import { Text } from "@/components/ui/text";
import { useTranslation } from "@/lib/i18n";
import { css } from "@/styled-system/css";
import { Box, Grid, HStack, LStack, styled } from "@/styled-system/jsx";

type Props = {
  initialTagList: TagListResult;
};

type SortMode = "popular" | "alphabetical";

export function TagsIndexScreen(props: Props) {
  const t = useTranslation();
  const [search, setSearch] = useState("");
  const [sort, setSort] = useState<SortMode>("popular");
  const { data, error } = useTagList(
    {},
    { swr: { fallbackData: props.initialTagList } },
  );
  if (!data) {
    return <Unready error={error} />;
  }

  const used = data.tags.filter((t) => t.item_count > 0);
  const totalItems = used.reduce((sum, t) => sum + t.item_count, 0);
  const maxCount = Math.max(1, ...used.map((t) => t.item_count));

  const query = search.toLowerCase().trim();
  const tags = used
    .filter((t) => t.name.toLowerCase().includes(query))
    .sort((a, b) =>
      sort === "popular"
        ? b.item_count - a.item_count || a.name.localeCompare(b.name)
        : a.name.localeCompare(b.name),
    );

  const formatCount = (n: number) =>
    n === 1
      ? t.tags.itemCountOne
      : t.tags.itemCountOther.replace("{count}", String(n));

  return (
    <LStack gap="6">
      <HStack gap="4" alignItems="center">
        <Box
          display="flex"
          alignItems="center"
          justifyContent="center"
          flexShrink="0"
          w="12"
          h="12"
          borderRadius="l3"
          bg="bg.subtle"
          borderWidth="thin"
          borderColor="border.subtle"
          color="fg.muted"
        >
          <TagIcon w="6" h="6" />
        </Box>
        <LStack gap="1">
          <Heading as="h1" size="2xl">
            {t.tags.indexTitle}
          </Heading>
          <Text textStyle="sm" color="fg.muted">
            {t.tags.indexSubtitle}
          </Text>
        </LStack>
      </HStack>

      <HStack gap="3" w="full" flexWrap={{ base: "wrap", md: "nowrap" }}>
        <InputGroup
          flex="1"
          minW="0"
          startElement={<SearchIcon w="4" h="4" color="fg.muted" />}
        >
          <Input
            placeholder={t.tags.searchPlaceholder}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
          />
        </InputGroup>

        <HStack
          gap="1"
          p="1"
          flexShrink="0"
          borderRadius="l2"
          borderWidth="thin"
          borderColor="border.subtle"
          bg="bg.subtle"
        >
          <SortButton
            active={sort === "popular"}
            onClick={() => setSort("popular")}
          >
            {t.tags.sortPopular}
          </SortButton>
          <SortButton
            active={sort === "alphabetical"}
            onClick={() => setSort("alphabetical")}
          >
            {t.tags.sortAlphabetical}
          </SortButton>
        </HStack>
      </HStack>

      <Text textStyle="xs" color="fg.muted" fontWeight="medium">
        {used.length} {t.tags.totalTags} · {totalItems}{" "}
        {t.tags.totalTaggedItems}
      </Text>

      {tags.length > 0 ? (
        <Grid
          w="full"
          gap="3"
          columns={{ base: 1, sm: 2, lg: 3, xl: 4 }}
        >
          {tags.map((tag) => (
            <TagCard
              key={tag.name}
              tag={tag}
              label={formatCount(tag.item_count)}
              ratio={tag.item_count / maxCount}
            />
          ))}
        </Grid>
      ) : (
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
            {query
              ? t.tags.noSearchResults.replace("{query}", search.trim())
              : t.tags.emptyState}
          </Text>
        </Box>
      )}
    </LStack>
  );
}

function SortButton({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <Button
      size="xs"
      variant={active ? "outline" : "ghost"}
      bg={active ? "bg.default" : undefined}
      color={active ? "fg.default" : "fg.muted"}
      aria-pressed={active}
      onClick={onClick}
    >
      {children}
    </Button>
  );
}

const cardStyles = css({
  position: "relative",
  display: "flex",
  flexDirection: "column",
  gap: "3",
  p: "4",
  overflow: "hidden",
  borderRadius: "l3",
  borderWidth: "thin",
  borderColor: "border.subtle",
  bg: "bg.default",
  transitionProperty: "common",
  transitionDuration: "fast",
  _hover: {
    borderColor: "colorPalette.border",
    bg: "bg.subtle",
    shadow: "sm",
  },
  _focusVisible: {
    borderColor: "colorPalette.border",
    shadow: "md",
  },
});

function TagCard({
  tag,
  label,
  ratio,
}: {
  tag: TagReference;
  label: string;
  ratio: number;
}) {
  return (
    <Link
      href={`/tags/${encodeURIComponent(tag.name)}`}
      className={cardStyles}
      style={badgeColourCSS(tag.colour)}
    >
      <HStack gap="2" alignItems="center" minW="0">
        <styled.span
          flexShrink="0"
          w="2.5"
          h="2.5"
          borderRadius="full"
          bg="colorPalette.border"
        />
        <Text
          fontWeight="semibold"
          textStyle="md"
          overflow="hidden"
          textOverflow="ellipsis"
          whiteSpace="nowrap"
        >
          {tag.name}
        </Text>
      </HStack>

      <LStack gap="1.5">
        <Text textStyle="xs" color="fg.muted">
          {label}
        </Text>
        <Box
          w="full"
          h="1"
          borderRadius="full"
          bg="bg.emphasized"
          overflow="hidden"
        >
          <Box
            h="full"
            borderRadius="full"
            bg="colorPalette.border"
            style={{ width: `${Math.max(ratio * 100, 4)}%` }}
          />
        </Box>
      </LStack>
    </Link>
  );
}
