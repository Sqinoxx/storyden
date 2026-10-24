"use client";

import { useMemo } from "react";

import { useCategoryList } from "@/api/openapi-client/categories";
import { Category } from "@/api/openapi-schema";

import { DiscussionRoute } from "@/components/site/Navigation/Anchors/Discussion";
import { BreadcrumbLink, BreadcrumbTrail } from "@/components/ui/breadcrumb-trail";
import { useTranslation } from "@/lib/i18n";

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

  return (
    <BreadcrumbTrail>
      <BreadcrumbLink href={DiscussionRoute}>{t.nav.discussion}</BreadcrumbLink>
      {ancestors.map((c) => (
        <BreadcrumbLink key={c.id} href={`/d/${c.slug}`}>
          {c.name}
        </BreadcrumbLink>
      ))}
      <BreadcrumbLink href={`/d/${category.slug}`} current>
        {category.name}
      </BreadcrumbLink>
    </BreadcrumbTrail>
  );
}
