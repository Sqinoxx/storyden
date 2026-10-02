import type { Translations } from "@/lib/i18n";

type KindKey = keyof Translations["drive"]["kinds"];

const KIND_LABELS: [RegExp, KindKey][] = [
  [/^application\/pdf$/, "pdf"],
  [/^image\//, "image"],
  [/^video\//, "video"],
  [/^audio\//, "audio"],
  [/^text\/csv$/, "csv"],
  [/^text\//, "text"],
  [/wordprocessingml|msword/, "document"],
  [/spreadsheetml|ms-excel/, "spreadsheet"],
  [/presentationml|ms-powerpoint/, "presentation"],
  [/zip|compressed|tar|rar|7z/, "archive"],
];

export function driveKindLabel(
  t: Translations,
  mimeType: string,
  isFolder: boolean,
) {
  if (isFolder) return t.drive.kinds.folder;

  const match = KIND_LABELS.find(([pattern]) => pattern.test(mimeType));

  return t.drive.kinds[match?.[1] ?? "file"];
}

const UNITS = ["B", "KB", "MB", "GB", "TB"];

export function driveFileSize(bytes: number | undefined) {
  if (bytes === undefined) return "";

  let value = bytes;
  let unit = 0;

  while (value >= 1024 && unit < UNITS.length - 1) {
    value /= 1024;
    unit++;
  }

  const rounded = unit === 0 ? value : Math.round(value * 10) / 10;

  return `${rounded} ${UNITS[unit]}`;
}

export function driveModifiedAt(timestamp: string | undefined) {
  if (!timestamp) return "";

  const date = new Date(timestamp);
  if (Number.isNaN(date.getTime())) return "";

  return date.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  });
}
