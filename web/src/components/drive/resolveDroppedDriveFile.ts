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
};

export const UPLOADABLE_DOCUMENT_MIMES = Object.keys(FALLBACK_EXTENSIONS);

const IMPORTABLE_EXTENSION = /\.(pdf|docx?|pptx?|xlsx?)$/i;

const DRIVE_ID = /^[A-Za-z0-9_-]{10,}$/;

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

  if (url.host === "drive.google.com" || url.host === "docs.google.com") {
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

export function readDropURL(dt: DataTransfer | null): string | undefined {
  if (!dt) return undefined;

  const uriList = dt
    .getData("text/uri-list")
    .split(/\r?\n/)
    .map((line) => line.trim())
    .find((line) => line !== "" && !line.startsWith("#"));

  if (uriList) return uriList;

  const text = dt.getData("text/plain").trim();
  if (!text || /\s/.test(text)) return undefined;

  return text;
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

  if (url.host === "drive.google.com" || url.host === "docs.google.com") {
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
