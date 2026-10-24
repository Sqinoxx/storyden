"use client";

import { Fragment, useMemo } from "react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { Category } from "@/api/openapi-schema";

import { DiscussionRoute } from "@/components/site/Navigation/Anchors/Discussion";
import { BreadcrumbIcon } from "@/components/ui/icons/Breadcrumb";
import { LinkButton } from "@/components/ui/link-button";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, styled } from "@/styled-system/jsx";

type Props = {
  category: Category;
};

export function useCategoryAncestors(
  category: Pick<Category, "id" | "parent"> | undefined,
) {
  const { data } = useCategoryList();
  const id = category?.id;
  const parent = category?.parent;

  return useMemo(() => {
    const byID = new Map((data?.categories ?? []).map((c) => [c.id, c]));
    const chain: Category[] = [];
    const seen = new Set<string>(id ? [id] : []);

    let parentID = parent;
    while (parentID && !seen.has(parentID)) {
      const next = byID.get(parentID);
      if (!next) break;
      seen.add(parentID);
      chain.unshift(next);
      parentID = next.parent;
    }

    return chain;
  }, [data, id, parent]);
}

export function CategoryBreadcrumbs({ category }: Props) {
  const t = useTranslation();
  const ancestors = useCategoryAncestors(category);

  const crumbs = [...ancestors, category];

  return (
    <HStack
      w="full"
      color="fg.subtle"
      overflowX="hidden"
      justifyContent="flex-end"
    >
      <LinkButton
        size="xs"
        variant="subtle"
        flexShrink="0"
        minW="min"
        href={DiscussionRoute}
      >
        {t.nav.discussion}
      </LinkButton>
      {crumbs.map((c) => (
        <Fragment key={c.id}>
          <Box flexShrink="0">
            <BreadcrumbIcon />
          </Box>
          <LinkButton
            size="xs"
            variant="subtle"
            flexShrink="0"
            maxW="64"
            overflow="hidden"
            href={`/d/${c.slug}`}
          >
            <styled.span
              overflowX="hidden"
              textWrap="nowrap"
              textOverflow="ellipsis"
            >
              {c.name}
            </styled.span>
          </LinkButton>
        </Fragment>
      ))}
      <Box flex="1" />
    </HStack>
  );
}
