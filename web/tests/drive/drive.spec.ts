import { expect, test } from "@playwright/test";

import { createAdmin, login } from "../access_key_admin_assignment";
import { dismissOnboarding, unique } from "../helpers";

import {
  MOCK_ROOT_FOLDER_ID,
  MOCK_SUB_FOLDER_ID,
  addFolder,
} from "./drive_helpers";

const PASSWORD = "TestPassword123!";

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
