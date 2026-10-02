"use client";

import {
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Legend,
  Line,
  LineChart,
  ResponsiveContainer,
  Tooltip,
  TooltipContentProps,
  XAxis,
  YAxis,
} from "recharts";

import { Box, Flex, Stack, styled } from "@/styled-system/jsx";

import { formatNumber } from "./format";

export const COLORS = {
  accounts: "var(--colors-green-9)",
  threads: "var(--colors-blue-9)",
  replies: "var(--colors-blue-11)",
  logins: "var(--colors-red-9)",
  activeAccounts: "var(--colors-amber-9)",
  assets: "var(--colors-amber-11)",
  likes: "var(--colors-red-11)",
  reacts: "var(--colors-accent-7)",
  categories: "var(--colors-accent-9)",
  tags: "var(--colors-green-11)",
} as const;

const AXIS_PROPS = {
  stroke: "var(--colors-fg-muted)",
  fontSize: 12,
  tickLine: false,
  axisLine: false,
} as const;

const GRID_STROKE = "var(--colors-border-subtle)";
const CURSOR_FILL = "var(--colors-bg-subtle)";

export type LineSeries = { key: string; name: string; color: string };

export function MultiLineChart({
  data,
  series,
  xKey = "label",
  legend = series.length > 1,
}: {
  data: object[];
  series: LineSeries[];
  xKey?: string;
  legend?: boolean;
}) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data} margin={{ top: 8, right: 8, left: 0, bottom: 0 }}>
        <CartesianGrid stroke={GRID_STROKE} vertical={false} />
        <XAxis dataKey={xKey} {...AXIS_PROPS} minTickGap={16} />
        <YAxis {...AXIS_PROPS} allowDecimals={false} width={36} />
        <Tooltip content={ChartTooltip} />
        {legend && (
          <Legend
            verticalAlign="top"
            height={32}
            iconType="circle"
            iconSize={8}
            wrapperStyle={{ fontSize: 12 }}
          />
        )}
        {series.map((s) => (
          <Line
            key={s.key}
            type="monotone"
            dataKey={s.key}
            name={s.name}
            stroke={s.color}
            strokeWidth={2}
            dot={false}
            activeDot={{ r: 4 }}
          />
        ))}
      </LineChart>
    </ResponsiveContainer>
  );
}

export function SimpleBarChart({
  data,
  xKey,
  name,
  color,
  horizontal = false,
  highlight,
}: {
  data: object[];
  xKey: string;
  name: string;
  color: string;
  horizontal?: boolean;
  highlight?: unknown;
}) {
  return (
    <ResponsiveContainer width="100%" height="100%">
      <BarChart
        data={data}
        layout={horizontal ? "vertical" : "horizontal"}
        margin={{ top: 8, right: 8, left: 0, bottom: 0 }}
      >
        <CartesianGrid
          stroke={GRID_STROKE}
          vertical={horizontal}
          horizontal={!horizontal}
        />
        {horizontal ? (
          <>
            <XAxis type="number" {...AXIS_PROPS} allowDecimals={false} />
            <YAxis
              type="category"
              dataKey={xKey}
              {...AXIS_PROPS}
              width={120}
              interval={0}
              tickFormatter={truncate}
            />
          </>
        ) : (
          <>
            <XAxis dataKey={xKey} {...AXIS_PROPS} minTickGap={4} />
            <YAxis {...AXIS_PROPS} allowDecimals={false} width={36} />
          </>
        )}
        <Tooltip content={ChartTooltip} cursor={{ fill: CURSOR_FILL }} />
        <Bar
          dataKey="count"
          name={name}
          fill={color}
          radius={horizontal ? [0, 4, 4, 0] : [4, 4, 0, 0]}
          maxBarSize={horizontal ? 20 : 40}
        >
          {highlight !== undefined &&
            data.map((row, i) => (
              <Cell
                key={i}
                fillOpacity={
                  (row as Record<string, unknown>)[xKey] === highlight
                    ? 1
                    : 0.45
                }
              />
            ))}
        </Bar>
      </BarChart>
    </ResponsiveContainer>
  );
}

export function Sparkline({
  values,
  color,
}: {
  values: number[];
  color: string;
}) {
  const data = values.map((v, i) => ({ i, v }));

  return (
    <ResponsiveContainer width="100%" height="100%">
      <LineChart data={data} margin={{ top: 2, right: 2, left: 2, bottom: 2 }}>
        <YAxis hide domain={[0, "dataMax"]} />
        <Line
          type="monotone"
          dataKey="v"
          stroke={color}
          strokeWidth={1.5}
          dot={false}
          isAnimationActive={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

function truncate(value: string): string {
  return value.length > 18 ? `${value.slice(0, 17)}…` : value;
}

export function ChartTooltip({ active, payload, label }: TooltipContentProps) {
  if (!active || !payload?.length) return null;

  return (
    <Box
      bgColor="bg.default"
      borderWidth="thin"
      borderColor="border.subtle"
      borderRadius="md"
      boxShadow="md"
      p="3"
      fontSize="xs"
    >
      {label !== undefined && label !== "" && (
        <styled.div fontWeight="semibold" color="fg.default" mb="1">
          {label}
        </styled.div>
      )}
      <Stack gap="1">
        {payload.map((entry) => (
          <Flex key={String(entry.dataKey)} alignItems="center" gap="2">
            <styled.span
              display="inline-block"
              w="2"
              h="2"
              borderRadius="full"
              flexShrink="0"
              style={{ backgroundColor: entry.color }}
            />
            <styled.span color="fg.muted">{entry.name}:</styled.span>
            <styled.span
              fontWeight="semibold"
              color="fg.default"
              fontVariantNumeric="tabular-nums"
            >
              {formatNumber(entry.value as number)}
            </styled.span>
          </Flex>
        ))}
      </Stack>
    </Box>
  );
}
