import { MAX_ASSET_UPLOAD_BYTES } from "@/utils/asset";

import { useSettings } from "./settings-client";

/**
 * The effective max upload size in bytes, sourced from the admin-configurable
 * `max_upload_size_mb` setting. Falls back to the historical 50MB default
 * while settings are still loading or if the server omitted the field.
 */
export function useMaxUploadSizeBytes(): number {
  const { settings } = useSettings();

  const mb = settings?.max_upload_size_mb;
  if (!mb || mb <= 0) {
    return MAX_ASSET_UPLOAD_BYTES;
  }

  return mb * 1024 * 1024;
}
