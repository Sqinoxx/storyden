import { Locator, Page, expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import { MOCK_PDF_FILE_ID, MOCK_ROOT_FOLDER_ID, addFolder } from "../drive/drive_helpers";
import { dismissOnboarding, pasteText, quickShareForm, unique } from "../helpers";

const PASSWORD = "TestPassword123!";

const VIEWPORTS = [
  { name: "phone", width: 375, height: 740 },
  { name: "tablet-portrait", width: 820, height: 1180 },
  { name: "tablet-landscape", width: 1180, height: 820 },
  { name: "desktop", width: 1440, height: 900 },
];

async function expectInsideViewport(page: Page, bar: Locator) {
  const viewport = page.viewportSize()!;
  const box = (await bar.boundingBox())!;

  expect(box.x).toBeGreaterThanOrEqual(0);
  expect(box.x + box.width).toBeLessThanOrEqual(viewport.width);

  for (const button of await bar.getByRole("button").all()) {
    await expect(button).toBeInViewport();
    const b = (await button.boundingBox())!;
    expect(b.x).toBeGreaterThanOrEqual(box.x - 1);
    expect(b.x + b.width).toBeLessThanOrEqual(box.x + box.width + 1);
  }
}

test.describe("Quick share link paste bar", () => {
  test("uploads pasted file links and fits every screen size", async ({
    page,
  }, testInfo) => {
    const suffix = unique("pastebar");
    const admin = `admin${suffix}`.replace(/-/g, "");
    const slug = `paste-bar-${suffix}`;
    const folderName = `Paste shelf ${suffix}`;

    await withAdminAccessKey(async ({ categoryCreate }) => {
      await categoryCreate({
        colour: "#3b82f6",
        description: `Paste bar target ${suffix}`,
        name: `Paste bar ${suffix}`,
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

    for (const viewport of VIEWPORTS) {
      await page.setViewportSize(viewport);
      await page.goto(`/d/${slug}`);
      await dismissOnboarding(page);

      const form = quickShareForm(page);
      const editor = form.locator(".ProseMirror");
      await expect(editor).toBeVisible({ timeout: 10000 });
      const bar = form.locator("[data-link-paste-bar]");

      await pasteText(editor, "https://www.zm-online.de/news/aktuelles");
      await expect(bar).toContainText("Show link as");
      await expectInsideViewport(page, bar);
      await form.screenshot({
        path: testInfo.outputPath(`${viewport.name}-link.png`),
      });

      await bar.getByRole("button", { name: "Text" }).click();
      await expect(bar).toBeHidden();
      await expect(
        editor.locator('a[href="https://www.zm-online.de/news/aktuelles"]'),
      ).toBeVisible();

      const origin = new URL(page.url()).origin;
      const fileURL = `${origin}/drive/${folderID}/${MOCK_PDF_FILE_ID}`;

      await editor.press("End");
      await editor.press("Enter");
      await pasteText(editor, fileURL);

      await expect(bar).toContainText("Inserted as file", {
        timeout: 15000,
      });
      await expect(editor.getByText("handout.pdf")).toBeVisible();
      await expectInsideViewport(page, bar);
      await form.screenshot({
        path: testInfo.outputPath(`${viewport.name}-file.png`),
      });

      await bar.getByRole("button", { name: "Link", exact: true }).click();
      await expect(bar).toBeHidden();
      await expect(editor.getByText("handout.pdf")).toBeHidden();
      await expect(editor.locator(`a[href="${fileURL}"]`)).toBeVisible();
    }
  });
});
