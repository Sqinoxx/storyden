import { assetImport } from "@/api/openapi-client/assets";
import { driveFolderList } from "@/api/openapi-client/drive";
import { Asset } from "@/api/openapi-schema";
import { API_ADDRESS, WEB_ADDRESS } from "@/config";

import { getDriveFileURL } from "./url";

const FALLBACK_EXTENSIONS: Record<string, string> = {
  "application/pdf": "pdf",
  "application/msword": "doc",
  "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
    "docx",
  "application/vnd.ms-powerpoint": "ppt",
  "application/vnd.openxmlformats-officedocument.presentationml.presentation":
    "pptx",
  "application/vnd.ms-excel": "xls",
  "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": "xlsx",
  "image/jpeg": "jpg",
  "image/png": "png",
  "image/gif": "gif",
  "image/webp": "webp",
  "image/heic": "heic",
  "image/heif": "heif",
  "image/avif": "avif",
};

export const UPLOADABLE_DOCUMENT_MIMES = Object.keys(FALLBACK_EXTENSIONS);

const IMPORTABLE_EXTENSION =
  /\.(pdf|docx?|pptx?|xlsx?|jpe?g|png|gif|webp|heic|heif|avif)$/i;

const URL_PATTERN = /https?:\/\/[^\s"'<>]+/g;

const DRIVE_ID = /^[A-Za-z0-9_-]{10,}$/;

const GOOGLE_FILE_HOSTS = new Set([
  "drive.google.com",
  "docs.google.com",
  "drive.usercontent.google.com",
]);

export type DriveFileRef = {
  folderID?: string;
  fileID: string;
};

function hostOf(address: string | undefined): string | undefined {
  if (!address) return undefined;
  try {
    return new URL(address).host;
  } catch {
    return undefined;
  }
}

function ownHosts(): Set<string> {
  const hosts = [
    hostOf(API_ADDRESS),
    hostOf(WEB_ADDRESS),
    typeof window !== "undefined" ? window.location.host : undefined,
  ].filter((h): h is string => !!h);

  return new Set(hosts);
}

function segmentsOf(url: URL): string[] {
  return url.pathname
    .split("/")
    .filter(Boolean)
    .map((s) => decodeURIComponent(s));
}

export function parseDriveDropURL(
  raw: string | undefined,
): DriveFileRef | null {
  if (!raw) return null;

  let url: URL;
  try {
    url = new URL(raw.trim());
  } catch {
    return null;
  }

  if (url.protocol !== "https:" && url.protocol !== "http:") {
    return null;
  }

  if (GOOGLE_FILE_HOSTS.has(url.host)) {
    const queryID = url.searchParams.get("id");
    if (queryID && DRIVE_ID.test(queryID)) {
      return { fileID: queryID };
    }

    const segments = segmentsOf(url);
    const d = segments.indexOf("d");
    const id = d >= 0 ? segments[d + 1] : undefined;
    if (id && DRIVE_ID.test(id)) {
      return { fileID: id };
    }

    return null;
  }

  if (!ownHosts().has(url.host)) {
    return null;
  }

  const segments = segmentsOf(url);

  const [api, drive, folders, folderID, files, fileID, content] = segments;
  if (
    api === "api" &&
    drive === "drive" &&
    folders === "folders" &&
    files === "files" &&
    content === "content" &&
    segments.length === 7 &&
    folderID &&
    fileID
  ) {
    return { folderID, fileID };
  }

  if (segments.length === 3 && segments[0] === "drive") {
    const [, folder, child] = segments;
    if (folder && child) {
      return { folderID: folder, fileID: child };
    }
  }

  return null;
}

function safeGetData(dt: DataTransfer, type: string): string {
  try {
    return dt.getData(type) ?? "";
  } catch {
    return "";
  }
}

function urlsIn(text: string): string[] {
  return (text.match(URL_PATTERN) ?? []).map((u) => u.replace(/[),.;]+$/, ""));
}

function directURL(dt: DataTransfer): string | undefined {
  const uriList = safeGetData(dt, "text/uri-list")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line !== "" && !line.startsWith("#"));

  if (uriList) return uriList;

  const text = safeGetData(dt, "text/plain").trim();
  if (!text || /\s/.test(text)) return undefined;

  return text;
}

// Only a dragged link or image counts, never a text selection that merely
// contains one, so dragging a paragraph into the editor still inserts it.
function htmlLinks(html: string): string[] {
  if (!html || typeof DOMParser === "undefined") return [];

  const body = new DOMParser().parseFromString(html, "text/html").body;
  const elements = Array.from(body.querySelectorAll("a[href], img[src]"));
  if (elements.length === 0 || elements.length > 2) return [];

  const linkedText = elements
    .filter((el) => el.tagName === "A")
    .map((el) => el.textContent ?? "")
    .join("");
  if ((body.textContent ?? "").trim() !== linkedText.trim()) return [];

  return elements.map(
    (el) => el.getAttribute("href") ?? el.getAttribute("src") ?? "",
  );
}

function plainLinks(text: string): string[] {
  const lines = text
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  if (lines.length > 3) return [];

  return lines.filter((line) => /^https?:\/\/\S+$/.test(line));
}

const STANDARD_DRAG_TYPES = new Set([
  "downloadurl",
  "text/uri-list",
  "text/x-moz-url",
  "text/html",
  "text/plain",
  "files",
]);

// Desktop browsers put a dragged file's link in different places depending on
// the source: Google Drive uses Chrome's DownloadURL ("mime:name:url"), images
// from other sites carry it in text/html, Firefox adds text/x-moz-url. A file
// link anywhere in the drag wins over the plain page link.
export function readDropURLs(dt: DataTransfer | null): string[] {
  if (!dt) return [];

  const candidates: string[] = [];
  const add = (url: string | undefined) => {
    const trimmed = url?.trim();
    if (trimmed && !candidates.includes(trimmed)) candidates.push(trimmed);
  };

  urlsIn(safeGetData(dt, "DownloadURL")).forEach(add);
  add(directURL(dt));
  add(safeGetData(dt, "text/x-moz-url").split(/\r?\n/)[0]);
  htmlLinks(safeGetData(dt, "text/html")).forEach(add);
  plainLinks(safeGetData(dt, "text/plain")).forEach(add);

  for (const type of Array.from(dt.types ?? [])) {
    if (!STANDARD_DRAG_TYPES.has(type.toLowerCase())) {
      urlsIn(safeGetData(dt, type)).forEach(add);
    }
  }

  return candidates;
}

export function isFileLink(url: string | undefined): url is string {
  return parseDriveDropURL(url) !== null || isImportableFileURL(url);
}

export function readDropURL(dt: DataTransfer | null): string | undefined {
  if (!dt) return undefined;

  const fileLink = readDropURLs(dt).find(isFileLink);
  if (fileLink) return fileLink;

  const types = Array.from(dt.types ?? []);
  if (types.length > 0 && !types.includes("Files")) {
    console.info("Drop without a recognised file link, types:", types);
  }

  return directURL(dt);
}

export type ResolvedFileLink = { file: File } | { asset: Asset };

export async function resolveFileLink(
  url: string,
  onImportFailed?: () => void,
): Promise<ResolvedFileLink | null> {
  const ref = parseDriveDropURL(url);
  if (ref) {
    const file = await fetchDriveDocument(ref).catch(() => null);
    if (file) {
      return { file };
    }
    if (ref.folderID) {
      return null;
    }
  }

  if (isImportableFileURL(url)) {
    const asset = await importFromURL(url);
    if (asset) {
      return { asset };
    }
    onImportFailed?.();
  }

  return null;
}

export function filenameFromContentDisposition(
  header: string | null,
): string | undefined {
  if (!header) return undefined;

  const extended = /filename\*\s*=\s*(?:UTF-8|utf-8)''([^;]+)/.exec(header);
  if (extended?.[1]) {
    try {
      return decodeURIComponent(extended[1].trim().replace(/^"|"$/g, ""));
    } catch {
      // fall through to the plain parameter
    }
  }

  const plain = /filename\s*=\s*("([^"]*)"|[^;]+)/.exec(header);
  const name = plain?.[2] ?? plain?.[1];

  return name?.trim() || undefined;
}

function isUploadableDocument(mime: string): boolean {
  return UPLOADABLE_DOCUMENT_MIMES.includes(mime);
}

async function fetchFromFolder(
  folderID: string,
  fileID: string,
): Promise<File | null> {
  const res = await fetch(getDriveFileURL(folderID, fileID, "attachment"), {
    credentials: "include",
  });

  if (!res.ok) return null;

  const mime = (res.headers.get("content-type") ?? "")
    .split(";")[0]!
    .trim()
    .toLowerCase();

  if (!isUploadableDocument(mime)) {
    await res.body?.cancel().catch(() => {});
    return null;
  }

  const name =
    filenameFromContentDisposition(res.headers.get("content-disposition")) ??
    `dokument.${FALLBACK_EXTENSIONS[mime]}`;

  const blob = await res.blob();

  return new File([blob], name, { type: mime });
}

export async function fetchDriveDocument(
  ref: DriveFileRef,
): Promise<File | null> {
  if (ref.folderID) {
    return fetchFromFolder(ref.folderID, ref.fileID);
  }

  const { folders } = await driveFolderList();

  for (const folder of folders) {
    const file = await fetchFromFolder(folder.id, ref.fileID).catch(() => null);
    if (file) return file;
  }

  return null;
}

export function isImportableFileURL(raw: string | undefined): boolean {
  if (!raw) return false;

  let url: URL;
  try {
    url = new URL(raw.trim());
  } catch {
    return false;
  }

  if (url.protocol !== "https:" && url.protocol !== "http:") {
    return false;
  }

  if (ownHosts().has(url.host)) {
    return false;
  }

  if (GOOGLE_FILE_HOSTS.has(url.host)) {
    return parseDriveDropURL(raw) !== null;
  }

  try {
    return IMPORTABLE_EXTENSION.test(decodeURIComponent(url.pathname));
  } catch {
    return IMPORTABLE_EXTENSION.test(url.pathname);
  }
}

export async function importFromURL(url: string): Promise<Asset | null> {
  try {
    return await assetImport({ url });
  } catch {
    return null;
  }
}
