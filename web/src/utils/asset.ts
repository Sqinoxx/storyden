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
 * Removes internal random prefixes (such as 20-character xid prefixes e.g. "d9n2uhh3fmss73bsumm0-")
 * and formats file extensions for display (e.g. "-pdf" -> ".pdf").
 */
export function getCleanFilename(filename?: string): string {
  if (!filename) return "";

  // Remove 20-character xid prefix if present (e.g. "d9n2uhh3fmss73bsumm0-foo" -> "foo")
  let clean = filename.replace(/^[a-z0-9]{20}-/i, "");

  // If the filename ends with a hyphenated extension like "-pdf", "-docx", etc., convert the last hyphen to a dot
  // (e.g. "belegungen-drucken-5-1-pdf" -> "belegungen-drucken-5-1.pdf")
  clean = clean.replace(
    /-(pdf|doc|docx|xls|xlsx|ppt|pptx|zip|rar|7z|png|jpg|jpeg|gif|svg|txt|csv)$/i,
    ".$1"
  );

  return clean;
}


