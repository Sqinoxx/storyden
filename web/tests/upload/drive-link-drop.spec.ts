import { expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import {
  MOCK_PDF_FILE_ID,
  MOCK_ROOT_FOLDER_ID,
  MOCK_TEXT_FILE_ID,
  addFolder,
} from "../drive/drive_helpers";
import { dismissOnboarding, dropURL, quickShareForm, unique } from "../helpers";

const PASSWORD = "TestPassword123!";

test.describe("Quick share drive link drop", () => {
  test("uploads a dropped drive pdf link and keeps other links", async ({
    page,
  }) => {
    const suffix = unique("drivedrop");
    const admin = `admin${suffix}`.replace(/-/g, "");
    const folderName = `Drop shelf ${suffix}`;
    const slug = `drive-drop-${suffix}`;

    await withAdminAccessKey(async ({ categoryCreate }) => {
      await categoryCreate({
        colour: "#3b82f6",
        description: `Drive link drop target ${suffix}`,
        name: `Drive drop ${suffix}`,
        slug,
      });
    });

    await createAdmin(page.context(), admin, PASSWORD);
    await login(page, admin, PASSWORD);
    await dismissOnboarding(page);

    await addFolder(page, folderName, MOCK_ROOT_FOLDER_ID);

    await page.goto("/drive");
    await page.getByRole("link", { name: folderName }).click();
    await page.waitForURL(/\/drive\/[^/]+$/);
    const folderID = new URL(page.url()).pathname.split("/").pop()!;

    await page.goto(`/d/${slug}`);
    await dismissOnboarding(page);

    const form = quickShareForm(page);
    const editor = form.locator(".ProseMirror");
    await expect(editor).toBeVisible({ timeout: 10000 });

    const origin = new URL(page.url()).origin;

    await dropURL(
      page,
      editor,
      `${origin}/drive/${folderID}/${MOCK_PDF_FILE_ID}`,
    );

    await expect(editor.getByText("handout.pdf")).toBeVisible({
      timeout: 15000,
    });
    await expect(editor).not.toContainText("/drive/");

    const textURL = `${origin}/drive/${folderID}/${MOCK_TEXT_FILE_ID}`;
    await dropURL(page, editor, textURL);

    await expect(editor.locator(`a[href="${textURL}"]`)).toBeVisible({
      timeout: 15000,
    });
  });
});
