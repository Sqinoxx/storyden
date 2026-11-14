"use client";

import {
  Activity,
  Cpu,
  Database,
  Gauge,
  HardDrive,
  MemoryStick,
} from "lucide-react";
import { type ReactNode, useMemo, useState } from "react";

import { useAdminServerMetrics } from "@/api/openapi-client/admin";
import {
  AdminServerMetricsRange,
  type ServerMetrics,
  type ServerMetricsPoint,
} from "@/api/openapi-schema";
import { Button } from "@/components/ui/button";
import { Box, Flex, Grid, HStack, Stack, styled } from "@/styled-system/jsx";

import { MultiLineChart } from "../StatisticsSettingsScreen/charts";
import { ChartCard, StatSection } from "../StatisticsSettingsScreen/layout";

import {
  LOAD_COLOR,
  RANGE_LABEL,
  formatBytes,
  formatClock,
  formatDecimal,
  formatPercent,
  formatRate,
  formatUptime,
  loadLevel,
  percentOf,
} from "./format";

const LIVE_REFRESH_MS = 2000;
const HISTORY_REFRESH_MS = 30000;

const SERIES_COLOR = {
  cpu: "var(--colors-blue-9)",
  mem: "var(--colors-amber-9)",
  in: "var(--colors-green-9)",
  out: "var(--colors-blue-9)",
  read: "var(--colors-amber-9)",
  write: "var(--colors-red-9)",
  rps: "var(--colors-green-9)",
  p95: "var(--colors-red-9)",
};

export function ServerSettingsScreen() {
  const [range, setRange] = useState<AdminServerMetricsRange>("live");
  const { data, error } = useAdminServerMetrics(
    { range },
    {
      swr: {
        refreshInterval:
          range === "live" ? LIVE_REFRESH_MS : HISTORY_REFRESH_MS,
      },
    },
  );

  return (
    <Stack gap="10" width="full">
      <Flex
        justifyContent="space-between"
        alignItems="flex-end"
        flexWrap="wrap"
        gap="4"
        borderBottomWidth="thin"
        borderColor="border.subtle"
        pb="4"
      >
        <Box>
          <styled.h2 fontSize="xl" fontWeight="bold" color="fg.default">
            Server-Auslastung
          </styled.h2>
          <styled.p fontSize="sm" color="fg.muted" mt="1">
            Live-Werte des Servers, alle 5 Sekunden gemessen. Der Verlauf
            beginnt bei jedem Neustart neu und reicht bis zu 24 Stunden zurück.
          </styled.p>
        </Box>
        <RangeToggle value={range} onChange={setRange} />
      </Flex>

      {error && !data ? (
        <styled.p color="fg.error">
          Messwerte konnten nicht geladen werden.
        </styled.p>
      ) : (
        <>
          <OverviewSection data={data} />
          <HistorySection data={data} range={range} />
          <DetailSection data={data} />
        </>
      )}
    </Stack>
  );
}

function RangeToggle({
  value,
  onChange,
}: {
  value: AdminServerMetricsRange;
  onChange: (r: AdminServerMetricsRange) => void;
}) {
  return (
    <HStack gap="0.5" p="0.5" borderRadius="md" bgColor="bg.subtle">
      {Object.values(AdminServerMetricsRange).map((r) => (
        <Button
          key={r}
          type="button"
          size="xs"
          variant={value === r ? "solid" : "ghost"}
          onClick={() => onChange(r)}
        >
          {RANGE_LABEL[r]}
        </Button>
      ))}
    </HStack>
  );
}

function OverviewSection({ data }: { data: ServerMetrics | undefined }) {
  const c = data?.current;
  const memPercent = c ? percentOf(c.mem_used, c.mem_total) : 0;
  const diskPercent = c ? percentOf(c.disk_used, c.disk_total) : 0;

  return (
    <StatSection title="Jetzt">
      <Grid
        columns={{ base: 1, sm: 2, xl: 3 }}
        gap="4"
        data-testid="server-kpis"
      >
        <GaugeCard
          label="Prozessor"
          icon={<Cpu size={16} />}
          value={c ? formatPercent(c.cpu_percent) : undefined}
          detail={
            c && data
              ? `${data.static.cores} Kerne · Last ${formatDecimal(c.load1, 2)}`
              : undefined
          }
          percent={c?.cpu_percent}
        />
        <GaugeCard
          label="Arbeitsspeicher"
          icon={<MemoryStick size={16} />}
          value={c ? formatPercent(memPercent) : undefined}
          detail={
            c
              ? `${formatBytes(c.mem_used)} von ${formatBytes(c.mem_total)}`
              : undefined
          }
          percent={c ? memPercent : undefined}
        />
        <GaugeCard
          label="Festplatte"
          icon={<HardDrive size={16} />}
          value={c ? formatPercent(diskPercent) : undefined}
          detail={
            c
              ? `${formatBytes(c.disk_total - c.disk_used)} frei von ${formatBytes(c.disk_total)}`
              : undefined
          }
          percent={c ? diskPercent : undefined}
        />
        <GaugeCard
          label="Anfragen pro Sekunde"
          icon={<Activity size={16} />}
          value={c ? formatDecimal(c.requests_per_sec) : undefined}
          detail={
            c
              ? c.errors > 0
                ? `${c.errors} Serverfehler in den letzten 5 s`
                : "keine Serverfehler"
              : undefined
          }
        />
        <GaugeCard
          label="Antwortzeit"
          icon={<Gauge size={16} />}
          value={c ? `${Math.round(c.avg_latency_ms)} ms` : undefined}
          detail={
            c
              ? c.requests_per_sec > 0
                ? `95 % schneller als ${c.p95_latency_ms} ms`
                : "gerade keine Anfragen"
              : undefined
          }
        />
        <GaugeCard
          label="Datenbank"
          icon={<Database size={16} />}
          value={
            c?.db_size_bytes !== undefined
              ? formatBytes(c.db_size_bytes)
              : undefined
          }
          detail={
            c
              ? c.db_total !== undefined
                ? `${c.db_active ?? 0} aktive von ${c.db_total} Verbindungen`
                : `${c.db_in_use} von ${c.db_open} Verbindungen belegt`
              : undefined
          }
        />
      </Grid>

      {c && c.cpu_per_core.length > 1 && <CoreBars values={c.cpu_per_core} />}
    </StatSection>
  );
}

function GaugeCard({
  label,
  icon,
  value,
  detail,
  percent,
}: {
  label: string;
  icon: ReactNode;
  value: string | undefined;
  detail: string | undefined;
  percent?: number;
}) {
  const color =
    percent === undefined ? undefined : LOAD_COLOR[loadLevel(percent)];

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
      <styled.div
        fontSize="2xl"
        fontWeight="bold"
        color="fg.default"
        lineHeight="tight"
        fontVariantNumeric="tabular-nums"
      >
        {value ?? "…"}
      </styled.div>
      {percent !== undefined && <LoadBar percent={percent} color={color} />}
      <styled.span fontSize="xs" color="fg.muted" minH="4">
        {detail}
      </styled.span>
    </Flex>
  );
}

export function LoadBar({
  percent,
  color,
}: {
  percent: number;
  color?: string;
}) {
  const clamped = Math.min(100, Math.max(0, percent));

  return (
    <Box
      h="1.5"
      w="full"
      borderRadius="full"
      bgColor="bg.subtle"
      overflow="hidden"
    >
      <Box
        h="full"
        borderRadius="full"
        transition="all"
        style={{
          width: `${clamped}%`,
          backgroundColor: color ?? LOAD_COLOR[loadLevel(clamped)],
        }}
      />
    </Box>
  );
}

function CoreBars({ values }: { values: number[] }) {
  return (
    <Flex
      direction="column"
      gap="2"
      p="4"
      borderRadius="lg"
      borderWidth="thin"
      borderColor="border.subtle"
      bgColor="bg.default"
    >
      <styled.span fontSize="sm" fontWeight="medium" color="fg.muted">
        Auslastung pro Kern
      </styled.span>
      <Grid columns={{ base: 2, md: 4 }} columnGap="4" rowGap="2">
        {values.map((v, i) => (
          <Flex key={i} alignItems="center" gap="2">
            <styled.span
              fontSize="xs"
              color="fg.muted"
              w="12"
              flexShrink="0"
              fontVariantNumeric="tabular-nums"
            >
              Kern {i + 1}
            </styled.span>
            <LoadBar percent={v} />
            <styled.span
              fontSize="xs"
              color="fg.default"
              w="10"
              textAlign="right"
              flexShrink="0"
              fontVariantNumeric="tabular-nums"
            >
              {formatPercent(v)}
            </styled.span>
          </Flex>
        ))}
      </Grid>
    </Flex>
  );
}

type ChartRow = {
  label: string;
  cpu: number;
  mem: number;
  netIn: number;
  netOut: number;
  diskRead: number;
  diskWrite: number;
  rps: number;
  p95: number;
};

function toRows(
  points: ServerMetricsPoint[],
  range: AdminServerMetricsRange,
): ChartRow[] {
  const round = (v: number) => Math.round(v * 10) / 10;

  return points.map((p) => ({
    label: formatClock(p.t, range),
    cpu: round(p.cpu),
    mem: round(p.mem),
    netIn: round(p.net_in / 1024),
    netOut: round(p.net_out / 1024),
    diskRead: round(p.disk_read / 1024),
    diskWrite: round(p.disk_write / 1024),
    rps: round(p.rps),
    p95: round(p.p95_ms),
  }));
}

const compactNumber = new Intl.NumberFormat("de-DE", {
  notation: "compact",
  maximumFractionDigits: 1,
});

const kbFormatter = (kb: number) => compactNumber.format(kb);

function HistorySection({
  data,
  range,
}: {
  data: ServerMetrics | undefined;
  range: AdminServerMetricsRange;
}) {
  const rows = useMemo(
    () => (data ? toRows(data.history, range) : []),
    [data, range],
  );
  const loading = !data;
  const empty = !loading && rows.length < 2;
  const animate = range !== "live";

  return (
    <StatSection
      title="Verlauf"
      description={
        range === "24h"
          ? "Minutenmittelwerte, Antwortzeit als Höchstwert der Minute."
          : "Ein Messpunkt alle 5 Sekunden."
      }
    >
      <Grid columns={{ base: 1, lg: 2 }} gap="4">
        <ChartCard
          title="Prozessor & Arbeitsspeicher"
          description="Auslastung in Prozent"
          loading={loading}
          empty={empty}
        >
          <MultiLineChart
            data={rows}
            yMax={100}
            animate={animate}
            series={[
              { key: "cpu", name: "CPU %", color: SERIES_COLOR.cpu },
              { key: "mem", name: "RAM %", color: SERIES_COLOR.mem },
            ]}
          />
        </ChartCard>
        <ChartCard
          title="Anfragen"
          description="Anfragen pro Sekunde an Website und API"
          loading={loading}
          empty={empty}
        >
          <MultiLineChart
            data={rows}
            animate={animate}
            series={[
              { key: "rps", name: "Anfragen/s", color: SERIES_COLOR.rps },
            ]}
          />
        </ChartCard>
        <ChartCard
          title="Antwortzeit"
          description="95 % der Anfragen waren schneller als dieser Wert (ms)"
          loading={loading}
          empty={empty}
        >
          <MultiLineChart
            data={rows}
            animate={animate}
            series={[{ key: "p95", name: "p95 ms", color: SERIES_COLOR.p95 }]}
          />
        </ChartCard>
        <ChartCard
          title="Netzwerk"
          description="Datenverkehr der Forum-App (KB/s)"
          loading={loading}
          empty={empty}
        >
          <MultiLineChart
            data={rows}
            animate={animate}
            yFormatter={kbFormatter}
            series={[
              { key: "netIn", name: "Eingang KB/s", color: SERIES_COLOR.in },
              { key: "netOut", name: "Ausgang KB/s", color: SERIES_COLOR.out },
            ]}
          />
        </ChartCard>
        <ChartCard
          title="Festplatte"
          description="Lesen und Schreiben auf dem ganzen Server (KB/s)"
          loading={loading}
          empty={empty}
        >
          <MultiLineChart
            data={rows}
            animate={animate}
            yFormatter={kbFormatter}
            series={[
              { key: "diskRead", name: "Lesen KB/s", color: SERIES_COLOR.read },
              {
                key: "diskWrite",
                name: "Schreiben KB/s",
                color: SERIES_COLOR.write,
              },
            ]}
          />
        </ChartCard>
      </Grid>
    </StatSection>
  );
}

function DetailSection({ data }: { data: ServerMetrics | undefined }) {
  if (!data) return null;

  const { current: c, static: s } = data;
  const appUptime =
    (new Date(c.time).getTime() - new Date(s.app_started_at).getTime()) / 1000;

  const items: { label: string; value: string }[] = [
    { label: "Prozessor", value: s.cpu_model || "unbekannt" },
    {
      label: "Betriebssystem",
      value:
        [s.os, s.kernel_version].filter(Boolean).join(" · ") || "unbekannt",
    },
    {
      label: "Systemlast (1/5/15 min)",
      value: [c.load1, c.load5, c.load15]
        .map((v) => formatDecimal(v, 2))
        .join(" / "),
    },
    {
      label: "Auslagerungsspeicher",
      value:
        c.swap_total > 0
          ? `${formatBytes(c.swap_used)} von ${formatBytes(c.swap_total)}`
          : "keiner",
    },
    ...(c.temperature_c !== undefined
      ? [
          {
            label: "CPU-Temperatur",
            value: `${Math.round(c.temperature_c)} °C`,
          },
        ]
      : []),
    { label: "Server läuft seit", value: formatUptime(c.host_uptime_seconds) },
    { label: "Forum läuft seit", value: formatUptime(Math.max(0, appUptime)) },
    {
      label: "Forum-Prozess",
      value: `${formatPercent(c.process_cpu_percent)} CPU · ${formatBytes(c.process_rss)} RAM`,
    },
    {
      label: "Go-Speicher / Goroutinen",
      value: `${formatBytes(c.heap_bytes)} · ${c.goroutines}`,
    },
    {
      label: "Netzwerk jetzt",
      value: `↓ ${formatRate(c.net_in_bps)} · ↑ ${formatRate(c.net_out_bps)}`,
    },
    {
      label: "Festplatte jetzt",
      value: `Lesen ${formatRate(c.disk_read_bps)} · Schreiben ${formatRate(c.disk_write_bps)}`,
    },
    ...(c.ocr_queue !== undefined
      ? [{ label: "OCR-Warteschlange", value: `${c.ocr_queue} Dateien` }]
      : []),
    {
      label: "Lokale KI (Ollama)",
      value: !c.ai_enabled
        ? "aus oder nicht erreichbar"
        : c.ai_models.length === 0
          ? "bereit, kein Modell geladen"
          : c.ai_models
              .map((m) => `${m.name} (${formatBytes(m.size_bytes)})`)
              .join(", "),
    },
  ];

  return (
    <StatSection title="Details">
      <Grid
        columns={{ base: 1, md: 2 }}
        columnGap="8"
        rowGap="3"
        px="5"
        py="4"
        borderRadius="lg"
        borderWidth="thin"
        borderColor="border.subtle"
        bgColor="bg.subtle"
      >
        {items.map((item) => (
          <Box key={item.label} minW="0">
            <styled.div fontSize="xs" color="fg.muted">
              {item.label}
            </styled.div>
            <styled.div
              fontSize="sm"
              fontWeight="medium"
              color="fg.default"
              fontVariantNumeric="tabular-nums"
              wordBreak="break-word"
            >
              {item.value}
            </styled.div>
          </Box>
        ))}
      </Grid>
    </StatSection>
  );
}
