import { Thread } from "@/api/openapi-schema";

import { useCategoryAncestors } from "@/components/category/CategoryBreadcrumbs";
import { BreadcrumbLink, BreadcrumbTrail } from "@/components/ui/breadcrumb-trail";
import { useTranslation } from "@/lib/i18n";

import { DiscussionRoute } from "../site/Navigation/Anchors/Discussion";

type Props = {
  thread?: Thread;
};

export function Breadcrumbs({ thread }: Props) {
  const t = useTranslation();
  const ancestors = useCategoryAncestors(thread?.category);
  const categories = thread?.category ? [...ancestors, thread.category] : [];

  return (
    <BreadcrumbTrail>
      <BreadcrumbLink href={DiscussionRoute}>{t.nav.discussion}</BreadcrumbLink>
      {categories.map((c) => (
        <BreadcrumbLink key={c.id} href={`/d/${c.slug}`}>
          {c.name}
        </BreadcrumbLink>
      ))}
      {thread && (
        <BreadcrumbLink href={`/t/${thread.slug}`} current>
          {thread.title}
        </BreadcrumbLink>
      )}
    </BreadcrumbTrail>
  );
}
