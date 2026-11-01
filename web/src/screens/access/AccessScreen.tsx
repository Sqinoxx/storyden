"use client";

import { Lock } from "lucide-react";
import { useMemo } from "react";

import { useAccessPassList } from "@/api/openapi-client/access";
import { useCategoryList } from "@/api/openapi-client/categories";
import { AccessPassListResult } from "@/api/openapi-schema";
import { AccessPassCard } from "@/components/access/AccessPassCard";
import { EmptyState } from "@/components/site/EmptyState";
import { Unready } from "@/components/site/Unready";
import { Heading } from "@/components/ui/heading";
import { useTranslation } from "@/lib/i18n";
import { LStack, styled } from "@/styled-system/jsx";

type Props = {
  initialPasses: AccessPassListResult;
};

export function AccessScreen({ initialPasses }: Props) {
  const t = useTranslation();
  const { data, error } = useAccessPassList({
    swr: { fallbackData: initialPasses },
  });
  const { data: categories } = useCategoryList();

  const namesByID = useMemo(
    () => new Map((categories?.categories ?? []).map((c) => [c.id, c.name])),
    [categories],
  );

  if (!data) return <Unready error={error} />;

  return (
    <LStack gap="4" w="full">
      <Heading size="lg">{t.access.title}</Heading>
      <styled.p color="fg.muted">{t.access.intro}</styled.p>

      {data.passes.length === 0 ? (
        <EmptyState
          icon={<Lock />}
          hideContributionLabel
          w="full"
          unauthenticatedLabel=""
        >
          <p>{t.access.noPasses}</p>
        </EmptyState>
      ) : (
        <LStack gap="3" w="full">
          {data.passes.map((p) => (
            <AccessPassCard
              key={p.id}
              pass={p}
              categoryNames={p.category_ids
                .map((id) => namesByID.get(id))
                .filter((n): n is string => Boolean(n))}
            />
          ))}
        </LStack>
      )}
    </LStack>
  );
}
