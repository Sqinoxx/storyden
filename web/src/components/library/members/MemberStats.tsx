"use client";

import { useProfileStats } from "@/api/openapi-client/profiles";
import { ProfileStatsResult } from "@/api/openapi-schema";
import { useTranslation } from "@/lib/i18n";
import { FINISHED_SEMESTER } from "@/lib/profile/academic";
import { Box, CardBox, HStack, VStack, styled } from "@/styled-system/jsx";

type Props = {
  initialStats?: ProfileStatsResult;
};

export function MemberStats({ initialStats }: Props) {
  const t = useTranslation();
  const { data } = useProfileStats({
    swr: { fallbackData: initialStats },
  });

  if (!data) {
    return null;
  }

  const fmt = new Intl.NumberFormat();
  const max = Math.max(1, ...data.by_semester.map((p) => p.count));

  function semesterLabel(semester: number) {
    if (semester === 0) return t.members.stats.unknownSemester;
    if (semester === FINISHED_SEMESTER) return t.members.stats.finishedSemester;
    return `${semester}.`;
  }

  return (
    <CardBox w="full">
      <VStack alignItems="stretch" gap="4">
        <HStack
          gap="6"
          flexWrap="wrap"
          alignItems="baseline"
          justifyContent="space-between"
        >
          <HStack gap="2" alignItems="baseline">
            <styled.span
              fontSize="3xl"
              fontWeight="bold"
              fontVariantNumeric="tabular-nums"
              lineHeight="tight"
            >
              {fmt.format(data.total)}
            </styled.span>
            <styled.span color="fg.muted">{t.members.stats.total}</styled.span>
          </HStack>

          <HStack gap="6" flexWrap="wrap">
            <Figure value={fmt.format(data.joined_30d)}>
              {t.members.stats.joined30d}
            </Figure>
            <Figure value={fmt.format(data.active_30d)}>
              {t.members.stats.active30d}
            </Figure>
          </HStack>
        </HStack>

        <VStack alignItems="stretch" gap="2">
          <styled.span fontSize="sm" color="fg.subtle" fontWeight="medium">
            {t.members.stats.bySemester}
          </styled.span>

          <HStack gap="1" alignItems="end" w="full">
            {data.by_semester.map((p) => (
              <VStack
                key={p.semester}
                flex="1"
                minW="0"
                gap="1"
                title={`${semesterLabel(p.semester)}: ${p.count}`}
              >
                <styled.span
                  fontSize="xs"
                  color={p.count === 0 ? "fg.subtle" : "fg.muted"}
                  fontVariantNumeric="tabular-nums"
                >
                  {p.count}
                </styled.span>
                <Box w="full" h="16" display="flex" alignItems="end">
                  <Box
                    w="full"
                    minH="0.5"
                    borderTopRadius="sm"
                    bg={p.count === 0 ? "bg.muted" : "accent.default"}
                    style={{ height: `${(p.count / max) * 100}%` }}
                  />
                </Box>
                <styled.span
                  fontSize="xs"
                  color="fg.subtle"
                  whiteSpace="nowrap"
                  overflow="hidden"
                  textOverflow="ellipsis"
                  maxW="full"
                >
                  {semesterLabel(p.semester)}
                </styled.span>
              </VStack>
            ))}
          </HStack>
        </VStack>
      </VStack>
    </CardBox>
  );
}

function Figure(props: React.PropsWithChildren<{ value: string }>) {
  return (
    <HStack gap="1.5" alignItems="baseline">
      <styled.span
        fontSize="lg"
        fontWeight="semibold"
        fontVariantNumeric="tabular-nums"
      >
        {props.value}
      </styled.span>
      <styled.span fontSize="sm" color="fg.muted">
        {props.children}
      </styled.span>
    </HStack>
  );
}
