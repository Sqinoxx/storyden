import { API_ADDRESS } from "@/config";

export function getAssetURL(filename?: string) {
  if (!filename) return "";
  if (filename.startsWith("http://") || filename.startsWith("https://")) {
    return filename;
  }
  const prefix = filename.startsWith("/") ? "" : "/";
  return `${API_ADDRESS}${prefix}${filename}`;
}

/**
 * Removes internal random prefixes (such as 20-character xid prefixes e.g. "d9n2uhh3fmss73bsumm0-" or "d9ihuvuqbbt08i08p1j0_")
 * and formats file extensions for display (e.g. "-pdf" -> ".pdf").
 */
export function getCleanFilename(filename?: string): string {
  if (!filename) return "";

  let clean = filename.trim();

  // Strip query/hash params and any leading path components
  clean = (clean.split("?")[0] ?? "").split("#")[0] ?? clean;
  clean = clean.split("/").pop() ?? clean;

  // If the filename ends with a hyphenated extension like "-pdf", "-docx", etc., convert the last hyphen to a dot
  // (e.g. "belegungen-drucken-5-1-pdf" -> "belegungen-drucken-5-1.pdf")
  clean = clean.replace(
    /-([a-zA-Z0-9]{1,6})$/,
    ".$1"
  );

  // Repeatedly remove random ID / xid / hex prefixes (e.g. 20-32 alphanumeric chars or UUIDs followed by -, _, or .)
  let prev = "";
  while (clean !== prev) {
    prev = clean;
    clean = clean.replace(
      /^([a-z0-9]{20,32}|[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12})[-_.]/i,
      ""
    );
  }

  // Convert hyphens to spaces for slugified filenames that lack spaces (e.g. "belegungen-drucken-1.pdf" -> "belegungen drucken 1.pdf")
  if (!clean.includes(" ")) {
    const extIndex = clean.lastIndexOf(".");
    const base = extIndex !== -1 ? clean.slice(0, extIndex) : clean;
    const ext = extIndex !== -1 ? clean.slice(extIndex) : "";
    clean = base.replace(/-/g, " ") + ext;
  }

  if (!clean) return filename;
  if (clean.startsWith(".")) return `file${clean}`;

  return clean;
}

/**
 * Normalizes an asset path or URL to its unique filename identifier.
 * E.g. "/api/assets/d9n2uhh3fmss73bsumm0-foo-pdf" -> "d9n2uhh3fmss73bsumm0-foo-pdf"
 * "assets/d9n2uhh3fmss73bsumm0-foo-pdf" -> "d9n2uhh3fmss73bsumm0-foo-pdf"
 */
export function normalizeAssetPath(pathOrUrl?: string | null): string {
  if (!pathOrUrl) return "";
  let clean = pathOrUrl.trim();
  try {
    if (clean.startsWith("http://") || clean.startsWith("https://")) {
      clean = new URL(clean).pathname;
    }
  } catch {}
  clean = (clean.split("?")[0] ?? "").split("#")[0] ?? "";
  clean = clean.replace(/^\/+/, "");
  clean = clean.replace(/^api\/assets\//, "").replace(/^assets\//, "");
  return clean.toLowerCase();
}

/**
 * Normalizes a filename for fuzzy identity matching across raw/slugified and clean display versions.
 * E.g. "wacker-standortunterweisung-pdf" -> "wackerstandortunterweisungpdf"
 * "Wacker Standortunterweisung.pdf" -> "wackerstandortunterweisungpdf"
 */
export function normalizeFilename(name?: string | null): string {
  if (!name) return "";
  const clean = getCleanFilename(name).toLowerCase();
  return clean.replace(/[^a-z0-9]/g, "");
}

/**
 * Fallback used only while the admin-configurable upload limit (see
 * useMaxUploadSizeBytes in @/lib/settings/uploads) hasn't loaded yet, or if
 * the server omits it. Mirrors the backend's MAX_UPLOAD_SIZE_MB default.
 */
export const MAX_ASSET_UPLOAD_BYTES = 50 * 1024 * 1024;

export function escapeHTML(value: string): string {
  return value
    .replace(/&/g, "&amp;")
    .replace(/</g, "&lt;")
    .replace(/>/g, "&gt;")
    .replace(/"/g, "&quot;")
    .replace(/'/g, "&#39;");
}

/**
 * Builds the editor markup for an uploaded attachment. Filenames are user
 * controlled and end up inside both attribute values and element text, so every
 * interpolation has to be escaped — an unescaped one lets a crafted filename
 * break out of the attribute and inject markup into the post body.
 */
export function buildAttachmentMarkup(assetPath: string, file: File): string {
  const url = escapeHTML(getAssetURL(assetPath));
  const name = escapeHTML(file.name);

  if (/^image\//i.test(file.type)) {
    return `<img src="${url}" alt="${name}" />`;
  }

  return `<a href="${url}" data-type="file-attachment" data-filename="${name}" download="${name}">${name}</a>`;
}

/**
 * Downloads an asset without opening a blank tab.
 *
 * Credentials are included because getAssetURL points at API_ADDRESS, which is a
 * different origin in any non-fullstack deployment — without them the session
 * cookie is dropped and private assets 401. Revoking the object URL is deferred
 * by a tick because Firefox and Safari abort a download whose blob URL is
 * revoked in the same task as the click.
 */
export async function downloadAsset(url: string, filename: string) {
  const click = (href: string) => {
    const link = document.createElement("a");
    link.href = href;
    link.download = filename;
    document.body.appendChild(link);
    link.click();
    link.remove();
  };

  try {
    const res = await fetch(url, { credentials: "include" });
    if (!res.ok) throw new Error(`Download failed with status ${res.status}`);

    const blobUrl = URL.createObjectURL(await res.blob());
    click(blobUrl);
    setTimeout(() => URL.revokeObjectURL(blobUrl), 0);
  } catch {
    click(url);
  }
}


