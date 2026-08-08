const KIND_LABELS: [RegExp, string][] = [
  [/^application\/pdf$/, "PDF"],
  [/^image\//, "Image"],
  [/^video\//, "Video"],
  [/^audio\//, "Audio"],
  [/^text\/csv$/, "CSV"],
  [/^text\//, "Text"],
  [/wordprocessingml|msword/, "Document"],
  [/spreadsheetml|ms-excel/, "Spreadsheet"],
  [/presentationml|ms-powerpoint/, "Presentation"],
  [/zip|compressed|tar|rar|7z/, "Archive"],
];

export function driveKindLabel(mimeType: string, isFolder: boolean) {
  if (isFolder) return "Folder";

  const match = KIND_LABELS.find(([pattern]) => pattern.test(mimeType));

  return match?.[1] ?? "File";
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
