import { LinkListResult } from "@/api/openapi-schema";

import { EmptyState } from "@/components/site/EmptyState";

import { LinkCardRows } from "../LinkCardList";
import { useTranslation } from "@/lib/i18n";

type Props = {
  links: LinkListResult;
  show?: number;
};

export function LinkResultList({ links, show }: Props) {
  const t = useTranslation();
  if (links.links.length === 0) {
    return <EmptyState hideContributionLabel>{t.library.noLinks}</EmptyState>;
  }

  const shown = show ? links.links.slice(0, show) : links.links;

  return <LinkCardRows links={shown} />;
}
