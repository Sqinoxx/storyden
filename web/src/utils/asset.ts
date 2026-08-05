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


