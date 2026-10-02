import type {
  AdminStatistics200,
  StatisticsSeriesPoint,
  StatisticsTrend,
} from "@/api/openapi-schema";
import { FINISHED_SEMESTER } from "@/lib/profile/academic";

export type Granularity = "daily" | "monthly" | "yearly";

export const GRANULARITY_LABEL: Record<Granularity, string> = {
  daily: "30 Tage",
  monthly: "12 Monate",
  yearly: "5 Jahre",
};

export type SeriesKey =
  | "accounts"
  | "threads"
  | "logins"
  | "activeAccounts"
  | "assets"
  | "likes"
  | "reacts";

const GRANULARITY_SUFFIX = {
  daily: "Daily",
  monthly: "Monthly",
  yearly: "Yearly",
} as const;

export function seriesFor(
  data: AdminStatistics200,
  key: SeriesKey,
  granularity: Granularity,
): StatisticsSeriesPoint[] {
  return data[`${key}${GRANULARITY_SUFFIX[granularity]}`];
}

export type SeriesRow = { label: string } & Partial<Record<SeriesKey, number>>;

export function mergeSeries(
  data: AdminStatistics200 | undefined,
  keys: SeriesKey[],
  granularity: Granularity,
): SeriesRow[] {
  if (!data || keys.length === 0) return [];

  const [first, ...rest] = keys.map((k) => seriesFor(data, k, granularity));

  return first!.map((point, i) => {
    const row: SeriesRow = { label: formatDate(point.date, granularity) };
    row[keys[0]!] = point.count;
    rest.forEach((series, j) => {
      row[keys[j + 1]!] = series[i]?.count ?? 0;
    });
    return row;
  });
}

export function sumSeries(...series: StatisticsSeriesPoint[][]): number[] {
  const [first = []] = series;
  return first.map((_, i) =>
    series.reduce((acc, s) => acc + (s[i]?.count ?? 0), 0),
  );
}

export function sumTrends(...trends: StatisticsTrend[]): StatisticsTrend {
  return trends.reduce(
    (acc, t) => ({
      current: acc.current + t.current,
      previous: acc.previous + t.previous,
    }),
    { current: 0, previous: 0 },
  );
}

export type Delta =
  | { kind: "up" | "down"; percent: number }
  | { kind: "new" }
  | { kind: "flat" };

export function computeDelta({ current, previous }: StatisticsTrend): Delta {
  if (current === previous) return { kind: "flat" };
  if (previous === 0) return { kind: "new" };

  const percent = Math.round(((current - previous) / previous) * 100);
  if (percent === 0) return { kind: "flat" };

  return { kind: percent > 0 ? "up" : "down", percent: Math.abs(percent) };
}

const numberFormat = new Intl.NumberFormat("de-DE");

export function formatNumber(value: number | undefined): string {
  return numberFormat.format(value ?? 0);
}

export function formatDate(date: string, granularity: Granularity): string {
  const parsed = new Date(`${date}T00:00:00Z`);

  switch (granularity) {
    case "daily":
      return parsed.toLocaleDateString("de-DE", {
        day: "2-digit",
        month: "2-digit",
        timeZone: "UTC",
      });
    case "monthly":
      return parsed.toLocaleDateString("de-DE", {
        month: "short",
        year: "2-digit",
        timeZone: "UTC",
      });
    case "yearly":
      return parsed.toLocaleDateString("de-DE", {
        year: "numeric",
        timeZone: "UTC",
      });
  }
}

export function formatDateTime(value: string): string {
  return new Date(value).toLocaleString("de-DE", {
    day: "2-digit",
    month: "2-digit",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export function formatFachsemester(semester: number): string {
  if (semester === 0) return "Unbekannt";
  if (semester === FINISHED_SEMESTER) return "Fertig";
  return `${semester}. Sem.`;
}

const WEEKDAY_LABELS = ["Mo", "Di", "Mi", "Do", "Fr", "Sa", "So"];
const WEEKDAY_NAMES = [
  "Montag",
  "Dienstag",
  "Mittwoch",
  "Donnerstag",
  "Freitag",
  "Samstag",
  "Sonntag",
];

export function formatWeekday(weekday: number): string {
  return WEEKDAY_LABELS[weekday - 1] ?? `${weekday}`;
}

export function formatWeekdayName(weekday: number): string {
  return WEEKDAY_NAMES[weekday - 1] ?? `${weekday}`;
}

export function peak<T extends { count: number }>(points: T[]): T | undefined {
  const top = points.reduce<T | undefined>(
    (best, p) => (!best || p.count > best.count ? p : best),
    undefined,
  );
  return top && top.count > 0 ? top : undefined;
}

export function isEmpty(points: { count: number }[]): boolean {
  return points.every((p) => p.count === 0);
}

export function buildFachsemesterInsight(
  points: { semester: number; count: number }[],
) {
  const known = points.filter(
    (p) => p.semester !== 0 && p.semester !== FINISHED_SEMESTER,
  );
  if (known.length === 0) return null;

  const most = known.reduce((a, b) => (b.count > a.count ? b : a));
  const least = known.reduce((a, b) => (b.count < a.count ? b : a));

  if (most.count === 0) return null;

  return {
    mostLabel: formatFachsemester(most.semester),
    mostCount: most.count,
    leastLabel: formatFachsemester(least.semester),
    leastCount: least.count,
  };
}
