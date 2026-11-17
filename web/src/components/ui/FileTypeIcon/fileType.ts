export type FileKind =
  | "pdf"
  | "word"
  | "excel"
  | "csv"
  | "powerpoint"
  | "archive"
  | "image"
  | "video"
  | "audio"
  | "text"
  | "code"
  | "file";

export type FileTypeInfo = {
  kind: FileKind;
  label: string;
};

const EXTENSION_KINDS: Record<string, FileKind> = {
  pdf: "pdf",
  doc: "word",
  docx: "word",
  docm: "word",
  dot: "word",
  dotx: "word",
  odt: "word",
  rtf: "word",
  pages: "word",
  xls: "excel",
  xlsx: "excel",
  xlsm: "excel",
  xlsb: "excel",
  xlt: "excel",
  xltx: "excel",
  ods: "excel",
  numbers: "excel",
  csv: "csv",
  tsv: "csv",
  ppt: "powerpoint",
  pptx: "powerpoint",
  pptm: "powerpoint",
  pps: "powerpoint",
  ppsx: "powerpoint",
  odp: "powerpoint",
  key: "powerpoint",
  zip: "archive",
  rar: "archive",
  "7z": "archive",
  tar: "archive",
  gz: "archive",
  tgz: "archive",
  bz2: "archive",
  xz: "archive",
  jpg: "image",
  jpeg: "image",
  png: "image",
  gif: "image",
  webp: "image",
  svg: "image",
  bmp: "image",
  tif: "image",
  tiff: "image",
  heic: "image",
  avif: "image",
  dcm: "image",
  stl: "image",
  mp4: "video",
  mov: "video",
  avi: "video",
  mkv: "video",
  webm: "video",
  m4v: "video",
  mp3: "audio",
  wav: "audio",
  m4a: "audio",
  ogg: "audio",
  flac: "audio",
  aac: "audio",
  txt: "text",
  md: "text",
  log: "text",
  json: "code",
  xml: "code",
  html: "code",
  htm: "code",
  css: "code",
  js: "code",
  ts: "code",
  py: "code",
  go: "code",
  sql: "code",
  yaml: "code",
  yml: "code",
};

const MIME_KINDS: [RegExp, FileKind][] = [
  [/^application\/pdf$/, "pdf"],
  [/wordprocessingml|msword|opendocument\.text|rtf/, "word"],
  [/spreadsheetml|ms-excel|opendocument\.spreadsheet/, "excel"],
  [/^text\/(csv|tab-separated-values)$/, "csv"],
  [/presentationml|ms-powerpoint|opendocument\.presentation/, "powerpoint"],
  [/zip|compressed|x-tar|x-rar|x-7z|gzip|x-bzip/, "archive"],
  [/^image\//, "image"],
  [/^video\//, "video"],
  [/^audio\//, "audio"],
  [/json|xml|javascript|x-sh|x-python/, "code"],
  [/^text\//, "text"],
];

const DEFAULT_LABELS: Record<FileKind, string> = {
  pdf: "PDF",
  word: "DOC",
  excel: "XLS",
  csv: "CSV",
  powerpoint: "PPT",
  archive: "ZIP",
  image: "IMG",
  video: "VID",
  audio: "AUD",
  text: "TXT",
  code: "CODE",
  file: "",
};

export function fileExtension(filename: string | undefined) {
  if (!filename) return "";
  const clean = filename.split(/[?#]/)[0] ?? "";
  const dot = clean.lastIndexOf(".");
  if (dot <= 0 || dot === clean.length - 1) return "";
  return clean.slice(dot + 1).toLowerCase();
}

export function getFileTypeInfo(
  mimeType: string | undefined,
  filename: string | undefined,
): FileTypeInfo {
  const ext = fileExtension(filename);
  const kind =
    EXTENSION_KINDS[ext] ??
    MIME_KINDS.find(([pattern]) => pattern.test(mimeType ?? ""))?.[1] ??
    "file";

  const label =
    ext && ext.length <= 4 && /^[a-z0-9]+$/.test(ext)
      ? ext.toUpperCase()
      : DEFAULT_LABELS[kind];

  return { kind, label };
}
