import { Page, expect, test } from "@playwright/test";

import { createAdmin, login } from "../access_key_admin_assignment";
import { dismissOnboarding, unique } from "../helpers";

const PASSWORD = "TestPassword123!";

// The e2e backend runs with GOOGLE_DRIVE_SERVICE_ACCOUNT_JSON=mock, which
// serves a fixed in-memory tree: "Mock Drive" holding notes.txt and a
// "Handouts" subfolder containing handout.pdf.
const MOCK_ROOT_FOLDER_ID = "mock-root-folder";
const MOCK_SUB_FOLDER_ID = "mock-sub-folder";

// A Drive folder may only be registered once, so a shared backend means an
// earlier test or a retry may already hold the one we want. Clear it first to
// keep the spec independent of execution order.
async function unregisterFolder(page: Page, driveFolderID: string) {
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

async function addFolder(page: Page, name: string, driveFolderID: string) {
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

test.describe("google drive browser", () => {
  test("admin registers a folder and members can browse into it", async ({
    page,
  }) => {
    const suffix = unique("drive");
    const admin = `admin${suffix}`;
    const folderName = `Course material ${suffix}`;

    await createAdmin(page.context(), admin, PASSWORD);
    await login(page, admin, PASSWORD);
    await dismissOnboarding(page);

    await addFolder(page, folderName, MOCK_ROOT_FOLDER_ID);

    await page.goto("/drive");
    const folderLink = page.getByRole("link", { name: folderName });
    await expect(folderLink).toBeVisible();

    await folderLink.click();
    await page.waitForURL(/\/drive\/[^/]+$/);

    await expect(
      page.getByRole("cell", { name: "notes.txt", exact: true }),
    ).toBeVisible();

    // Descending keeps a trail back to the registered folder, which carries
    // the administrator's name rather than the folder's name inside Drive.
    const subfolder = page.getByRole("link", { name: "Handouts" });
    await expect(subfolder).toBeVisible();
    await subfolder.click();

    await expect(
      page.getByRole("cell", { name: "handout.pdf", exact: true }),
    ).toBeVisible();
    await expect(page.getByRole("link", { name: folderName })).toBeVisible();
  });

  test("restricted folders are not offered to signed-out visitors", async ({
    page,
    browser,
  }) => {
    const suffix = unique("drivehidden");
    const admin = `admin${suffix}`;
    const folderName = `Staff shelf ${suffix}`;

    await createAdmin(page.context(), admin, PASSWORD);
    await login(page, admin, PASSWORD);
    await dismissOnboarding(page);

    await addFolder(page, folderName, MOCK_SUB_FOLDER_ID);

    const row = page.getByRole("row", { name: new RegExp(folderName) });
    await row
      .getByRole("combobox", { name: `Visibility for ${folderName}` })
      .click();
    await page.getByRole("option", { name: "Admins only" }).click();
    await expect(row.getByText("Admins only")).toBeVisible();

    const visitorContext = await browser.newContext();

    try {
      const visitorPage = await visitorContext.newPage();
      await visitorPage.goto("/drive");

      await expect(
        visitorPage.getByRole("link", { name: folderName }),
      ).toBeHidden();
    } finally {
      await visitorContext.close();
    }
  });
});
