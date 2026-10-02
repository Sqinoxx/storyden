"use client";

import { ArrowDownRight, ArrowUpRight, Minus, Sparkles } from "lucide-react";
import type { ReactNode } from "react";

import type { StatisticsTrend } from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { Box, Flex, HStack, styled } from "@/styled-system/jsx";

import { Sparkline } from "./charts";
import {
  GRANULARITY_LABEL,
  Granularity,
  computeDelta,
  formatNumber,
} from "./format";

export function StatSection({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <styled.section display="flex" flexDirection="column" gap="4">
      <Box>
        <styled.h3
          fontSize="xs"
          fontWeight="semibold"
          color="fg.muted"
          textTransform="uppercase"
          letterSpacing="wide"
        >
          {title}
        </styled.h3>
        {description && (
          <styled.p fontSize="sm" color="fg.subtle" mt="0.5">
            {description}
          </styled.p>
        )}
      </Box>
      {children}
    </styled.section>
  );
}

export function ChartCard({
  title,
  description,
  actions,
  insight,
  footnote,
  loading,
  empty,
  height = "64",
  children,
}: {
  title: string;
  description?: string;
  actions?: ReactNode;
  insight?: ReactNode;
  footnote?: ReactNode;
  loading?: boolean;
  empty?: boolean;
  height?: "48" | "56" | "64" | "72" | "auto";
  children: ReactNode;
}) {
  return (
    <Flex
      direction="column"
      gap="4"
      p="5"
      minW="0"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.default"
    >
      <Flex
        justifyContent="space-between"
        alignItems="flex-start"
        flexWrap="wrap"
        gap="3"
      >
        <Box minW="0">
          <styled.h4 fontSize="md" fontWeight="semibold" color="fg.default">
            {title}
          </styled.h4>
          {description && (
            <styled.p fontSize="sm" color="fg.muted">
              {description}
            </styled.p>
          )}
        </Box>
        {actions}
      </Flex>

      {insight && !loading && !empty && (
        <styled.p
          fontSize="sm"
          color="fg.default"
          bgColor="bg.subtle"
          borderRadius="md"
          px="3"
          py="2"
        >
          {insight}
        </styled.p>
      )}

      <Box height={height} width="full">
        {loading ? (
          <Placeholder>Lade Statistiken…</Placeholder>
        ) : empty ? (
          <Placeholder>Noch keine Daten vorhanden.</Placeholder>
        ) : (
          children
        )}
      </Box>

      {footnote && (
        <styled.p fontSize="xs" color="fg.muted">
          {footnote}
        </styled.p>
      )}
    </Flex>
  );
}

function Placeholder({ children }: { children: ReactNode }) {
  return (
    <Flex
      height="full"
      minH="24"
      width="full"
      alignItems="center"
      justifyContent="center"
      borderRadius="md"
      bgColor="bg.subtle"
    >
      <styled.span fontSize="sm" color="fg.muted">
        {children}
      </styled.span>
    </Flex>
  );
}

export function GranularityToggle({
  value,
  onChange,
}: {
  value: Granularity;
  onChange: (g: Granularity) => void;
}) {
  return (
    <HStack gap="0.5" p="0.5" borderRadius="md" bgColor="bg.subtle">
      {(Object.keys(GRANULARITY_LABEL) as Granularity[]).map((g) => (
        <Button
          key={g}
          type="button"
          size="xs"
          variant={value === g ? "solid" : "ghost"}
          onClick={() => onChange(g)}
        >
          {GRANULARITY_LABEL[g]}
        </Button>
      ))}
    </HStack>
  );
}

export function KpiCard({
  label,
  icon,
  color,
  trend,
  sparkline,
  loading,
}: {
  label: string;
  icon: ReactNode;
  color: string;
  trend: StatisticsTrend | undefined;
  sparkline?: number[];
  loading: boolean;
}) {
  return (
    <Flex
      direction="column"
      gap="2"
      p="4"
      minW="0"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.default"
    >
      <Flex alignItems="center" justifyContent="space-between" gap="2">
        <styled.span fontSize="sm" fontWeight="medium" color="fg.muted">
          {label}
        </styled.span>
        <Flex
          alignItems="center"
          justifyContent="center"
          w="7"
          h="7"
          borderRadius="md"
          bgColor="bg.subtle"
          flexShrink="0"
        >
          {icon}
        </Flex>
      </Flex>

      <Flex alignItems="flex-end" justifyContent="space-between" gap="3">
        <Box minW="0">
          <styled.div
            fontSize="2xl"
            fontWeight="bold"
            color="fg.default"
            lineHeight="tight"
            fontVariantNumeric="tabular-nums"
          >
            {loading || !trend ? "…" : formatNumber(trend.current)}
          </styled.div>
          {!loading && trend && <DeltaBadge trend={trend} />}
        </Box>
        {!loading && sparkline && sparkline.some((v) => v > 0) && (
          <Box w="24" h="10" flexShrink="0">
            <Sparkline values={sparkline} color={color} />
          </Box>
        )}
      </Flex>
    </Flex>
  );
}

function DeltaBadge({ trend }: { trend: StatisticsTrend }) {
  const delta = computeDelta(trend);

  const { icon, text, color } = (() => {
    switch (delta.kind) {
      case "up":
        return {
          icon: <ArrowUpRight size={14} />,
          text: `+${delta.percent} %`,
          color: "green.11" as const,
        };
      case "down":
        return {
          icon: <ArrowDownRight size={14} />,
          text: `−${delta.percent} %`,
          color: "red.11" as const,
        };
      case "new":
        return {
          icon: <Sparkles size={14} />,
          text: "neu",
          color: "green.11" as const,
        };
      case "flat":
        return {
          icon: <Minus size={14} />,
          text: "±0 %",
          color: "fg.muted" as const,
        };
    }
  })();

  return (
    <Flex alignItems="center" gap="1" mt="1" fontSize="xs" flexWrap="wrap">
      <Flex alignItems="center" gap="0.5" color={color} fontWeight="semibold">
        {icon}
        {text}
      </Flex>
      <styled.span color="fg.muted">
        vs. {formatNumber(trend.previous)} davor
      </styled.span>
    </Flex>
  );
}

export function TotalsStrip({
  items,
  loading,
}: {
  items: { label: string; value: number | undefined }[];
  loading: boolean;
}) {
  return (
    <Flex
      flexWrap="wrap"
      rowGap="3"
      columnGap="6"
      px="5"
      py="4"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.subtle"
    >
      {items.map((item) => (
        <Box key={item.label}>
          <styled.div fontSize="xs" color="fg.muted">
            {item.label}
          </styled.div>
          <styled.div
            fontSize="md"
            fontWeight="semibold"
            color="fg.default"
            fontVariantNumeric="tabular-nums"
          >
            {loading ? "…" : formatNumber(item.value)}
          </styled.div>
        </Box>
      ))}
    </Flex>
  );
}
