import { API_ADDRESS } from "@/config";

/**
 * Content is proxied through Storyden rather than linked to Google so that
 * private folders work, and so members never see a drive.google.com URL.
 */
export function getDriveFileURL(
  folderID: string,
  fileID: string,
  disposition: "inline" | "attachment",
) {
  return `${API_ADDRESS}/api/drive/folders/${folderID}/files/${encodeURIComponent(fileID)}/content?disposition=${disposition}`;
}

export function getDriveFolderHref(folderID: string, childID?: string) {
  if (!childID) {
    return `/drive/${folderID}`;
  }

  return `/drive/${folderID}/${encodeURIComponent(childID)}`;
}
