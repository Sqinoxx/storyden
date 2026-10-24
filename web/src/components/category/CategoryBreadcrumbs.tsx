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

export function CategoryBreadcrumbs({ category }: Props) {
  const t = useTranslation();
  const { data } = useCategoryList();

  const ancestors = useMemo(() => {
    const byID = new Map((data?.categories ?? []).map((c) => [c.id, c]));
    const chain: Category[] = [];
    const seen = new Set<string>([category.id]);

    let parentID = category.parent;
    while (parentID && !seen.has(parentID)) {
      const parent = byID.get(parentID);
      if (!parent) break;
      seen.add(parentID);
      chain.unshift(parent);
      parentID = parent.parent;
    }

    return chain;
  }, [data, category.id, category.parent]);

  const crumbs = [...ancestors, category];

  return (
    <HStack
      w="full"
      color="fg.subtle"
      overflowX="scroll"
      pt="scrollGutter"
      mt="-scrollGutter"
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
    </HStack>
  );
}
