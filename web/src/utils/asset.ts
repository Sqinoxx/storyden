import { API_ADDRESS } from "@/config";

export function getAssetURL(filename?: string) {
  if (!filename) return "";
  if (filename.startsWith("http://") || filename.startsWith("https://")) {
    return filename;
  }
  const prefix = filename.startsWith("/") ? "" : "/";
  return `${API_ADDRESS}${prefix}${filename}`;
}

