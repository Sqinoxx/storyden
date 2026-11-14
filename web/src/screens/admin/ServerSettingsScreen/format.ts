import type { AdminServerMetricsRange } from "@/api/openapi-schema";

const UNITS = ["B", "KB", "MB", "GB", "TB"];

export function formatBytes(bytes: number | undefined, digits = 1): string {
  let value = bytes ?? 0;
  let unit = 0;
  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024;
    unit++;
  }
  const fraction = unit === 0 ? 0 : digits;
  return `${value.toLocaleString("de-DE", {
    minimumFractionDigits: fraction,
    maximumFractionDigits: fraction,
  })} ${UNITS[unit]}`;
}

export function formatRate(bytesPerSecond: number | undefined): string {
  return `${formatBytes(bytesPerSecond)}/s`;
}

export function formatDecimal(value: number | undefined, digits = 1): string {
  return (value ?? 0).toLocaleString("de-DE", {
    minimumFractionDigits: digits,
    maximumFractionDigits: digits,
  });
}

export function formatPercent(value: number | undefined): string {
  return `${Math.round(value ?? 0)} %`;
}

export function formatUptime(seconds: number): string {
  const days = Math.floor(seconds / 86400);
  const hours = Math.floor((seconds % 86400) / 3600);
  const minutes = Math.floor((seconds % 3600) / 60);

  if (days > 0) return `${days} T ${hours} Std`;
  if (hours > 0) return `${hours} Std ${minutes} Min`;
  return `${minutes} Min`;
}

export function formatClock(
  iso: string,
  range: AdminServerMetricsRange,
): string {
  return new Date(iso).toLocaleTimeString("de-DE", {
    hour: "2-digit",
    minute: "2-digit",
    second: range === "24h" ? undefined : "2-digit",
  });
}

export function percentOf(used: number, total: number): number {
  return total > 0 ? (used / total) * 100 : 0;
}

export type LoadLevel = "ok" | "warn" | "critical";

export function loadLevel(percent: number): LoadLevel {
  if (percent >= 90) return "critical";
  if (percent >= 70) return "warn";
  return "ok";
}

export const LOAD_COLOR: Record<LoadLevel, string> = {
  ok: "var(--colors-green-9)",
  warn: "var(--colors-amber-9)",
  critical: "var(--colors-red-9)",
};

export const RANGE_LABEL: Record<AdminServerMetricsRange, string> = {
  live: "Live",
  "1h": "1 Std",
  "24h": "24 Std",
};
