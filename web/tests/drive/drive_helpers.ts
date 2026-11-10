import { Page, expect } from "@playwright/test";

// The e2e backend runs with GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON=mock, which
// serves a fixed in-memory tree: "Mock Drive" holding notes.txt and a
// "Handouts" subfolder containing handout.pdf.
export const MOCK_ROOT_FOLDER_ID = "mock-root-folder";
export const MOCK_SUB_FOLDER_ID = "mock-sub-folder";

// A Drive folder may only be registered once, so a shared backend means an
// earlier test or a retry may already hold the one we want. Clear it first to
// keep the spec independent of execution order.
export async function unregisterFolder(page: Page, driveFolderID: string) {
  const row = page.getByRole("row").filter({
    has: page.getByText(driveFolderID, { exact: true }),
  });

  while ((await row.count()) > 0) {
    const first = row.first();
    await first.getByRole("button", { name: "Delete" }).click();
    await first.getByRole("button", { name: /are you sure/i }).click();
    await expect(first).toBeHidden();
  }
}

export async function addFolder(
  page: Page,
  name: string,
  driveFolderID: string,
) {
  await page.goto("/admin?tab=drive");
  await expect(
    page.getByRole("heading", { name: "Add a folder" }),
  ).toBeVisible();

  await unregisterFolder(page, driveFolderID);

  await page.getByRole("textbox", { name: "Name", exact: true }).fill(name);
  await page
    .getByRole("textbox", { name: "Google Drive folder link" })
    .fill(driveFolderID);
  await page.getByRole("button", { name: "Add", exact: true }).click();

  await expect(page.getByRole("cell", { name, exact: true })).toBeVisible();
}
