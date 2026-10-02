"use client";

import { useCategoryGet } from "@/api/openapi-client/categories";
import {
  CategoryGetOKResponse,
  Permission,
  ThreadListOKResponse,
} from "@/api/openapi-schema";
import { useSession } from "@/auth";
import Link from "next/link";

import { CategoryLayout } from "@/components/category/CategoryIndex/CategoryCardLayout";
import { CategoryMenu } from "@/components/category/CategoryMenu/CategoryMenu";
import { UnreadyBanner } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { LinkIcon } from "@/components/ui/icons/Link";
import { useTranslation } from "@/lib/i18n";
import { Box, HStack, LStack, WStack, styled } from "@/styled-system/jsx";
import { getAssetURL } from "@/utils/asset";
import { hasPermission } from "@/utils/permissions";

import { ThreadFeedScreen } from "../feed/ThreadFeedScreen/ThreadFeedScreen";

export type Props = {
  initialCategory: CategoryGetOKResponse;
  initialThreadList: ThreadListOKResponse;
  slug: string;
};

export function useCategoryScreen({ initialCategory, slug }: Props) {
  const session = useSession();

  const { data, error } = useCategoryGet(slug, {
    swr: { fallbackData: initialCategory },
  });

  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  const canEditCategory = hasPermission(session, Permission.MANAGE_CATEGORIES);
  const canPostAnywhere = hasPermission(
    session,
    Permission.POST_IN_ANY_CATEGORY,
  );

  // MANAGE_CATEGORIES no longer implies POST_IN_ANY_CATEGORY — the two are
  // granted independently, so normal users may only post in leaf categories.
  const isLeafCategory = !data.children || data.children.length === 0;

  const showQuickShare = canPostAnywhere || isLeafCategory;

  return {
    ready: true as const,
    data: {
      canEditCategory,
      showQuickShare,
      category: data,
    },
  };
}

type ScreenProps = {
  initialPage: number;
} & Props;

export function CategoryScreen(props: ScreenProps) {
  const t = useTranslation();
  const { ready, data, error } = useCategoryScreen(props);
  if (!ready) {
    return <UnreadyBanner error={error} />;
  }

  const { category, showQuickShare } = data;
  const coverImageURL = getAssetURL(category.cover_image?.path);

  return (
    <LStack>
      {coverImageURL && (
        <Box height="auto" width="full">
          <styled.img
            src={coverImageURL}
            alt="" // No alt image, decorative
            aria-hidden="true"
            width="full"
            height="full"
            borderRadius="md"
            objectFit="cover"
            objectPosition="center"
          />
        </Box>
      )}

      <LStack gap="1">
        <WStack alignItems="center">
          <Heading size="2xl" fontWeight="bold">
            {category.name}
          </Heading>

          <CategoryMenu category={category} />
        </WStack>

        <styled.p color="fg.muted">{category.description}</styled.p>
      </LStack>

      {category.children && category.children.length > 0 && (
        <LStack gap="1">
          <Heading size="sm" color="fg.muted">
            {t.category.subcategoriesHeading}
          </Heading>
          <CategoryLayout layout="grid" categories={category.children} />
        </LStack>
      )}

      {category.related && category.related.length > 0 && (
        <LStack gap="1">
          <Heading size="sm" color="fg.muted">
            {t.category.relatedHeading}
          </Heading>
          <HStack gap="2" flexWrap="wrap">
            {category.related.map((related) => (
              <Link
                key={related.id}
                href={`/d/${related.slug}`}
                title={related.description}
              >
                <HStack
                  gap="2"
                  px="3"
                  py="2"
                  borderWidth="thin"
                  borderColor="border.default"
                  borderRadius="md"
                  bg="bg.subtle"
                  color="fg.default"
                  fontWeight="medium"
                  transition="all"
                  _hover={{
                    bg: "bg.muted",
                    borderColor: "border.accent",
                  }}
                >
                  <LinkIcon w="4" h="4" />
                  {related.name}
                </HStack>
              </Link>
            ))}
          </HStack>
        </LStack>
      )}

      <ThreadFeedScreen
        initialPage={props.initialPage}
        initialPageData={props.initialThreadList}
        category={category}
        paginationBasePath={`/d/${data.category.slug}`}
        showCategorySelect={(category.children?.length ?? 0) > 0}
        hideCategoryBadge={true}
        enableSemesterGrouping={true}
        showQuickShare={showQuickShare}
      />
    </LStack>
  );
}
