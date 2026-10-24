import { Fragment } from "react";

import {
  CategoryReference,
  Thread,
  ThreadReference,
} from "@/api/openapi-schema";

import { useCategoryAncestors } from "@/components/category/CategoryBreadcrumbs";
import { BreadcrumbIcon } from "@/components/ui/icons/Breadcrumb";
import { LinkButton } from "@/components/ui/link-button";
import { Box, HStack, styled } from "@/styled-system/jsx";

import { DiscussionRoute } from "../site/Navigation/Anchors/Discussion";
import { useTranslation } from "@/lib/i18n";

type Props = {
  thread?: Thread;
};

type Breadcrumb =
  | {
      type: "category";
      key: string;
      category: CategoryReference;
    }
  | {
      type: "thread";
      key: string;
      thread: ThreadReference;
    };

export function Breadcrumbs({ thread }: Props) {
  const t = useTranslation();
  const ancestors = useCategoryAncestors(thread?.category);
  const category = thread?.category
    ? [...ancestors, thread.category].map((c) => ({
        type: "category" as const,
        key: c.id,
        category: c,
      }))
    : [];

  const crumbs: Breadcrumb[] = thread
    ? [
        ...category,
        {
          type: "thread",
          key: thread.id,
          thread: thread,
        },
      ]
    : [];

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
      {crumbs.map((c) => {
        return (
          <Fragment key={c.key}>
            <Box flexShrink="0">
              <BreadcrumbIcon />
            </Box>

            <BreadcrumbButton breadcrumb={c} />
          </Fragment>
        );
      })}
      <Box flex="1" />
    </HStack>
  );
}

function BreadcrumbButton({ breadcrumb }: { breadcrumb: Breadcrumb }) {
  switch (breadcrumb.type) {
    case "category":
      // TODO: Explore using the CategoryBadge component with subtle colour.
      return (
        <LinkButton
          size="xs"
          variant="subtle"
          flexShrink="0"
          maxW="64"
          overflow="hidden"
          href={`/d/${breadcrumb.category.slug}`}
        >
          {breadcrumb.category.name}
        </LinkButton>
      );

    case "thread":
      return (
        <LinkButton
          size="xs"
          variant="subtle"
          flexShrink="0"
          maxW="64"
          overflow="hidden"
          href={`/t/${breadcrumb.thread.slug}`}
        >
          <styled.span
            overflowX="hidden"
            textWrap="nowrap"
            textOverflow="ellipsis"
          >
            {breadcrumb.thread.title}
          </styled.span>
        </LinkButton>
      );
  }
}
