import { expect, test } from "@playwright/test";

import {
  createThread,
  ensureThreadCategory,
  registerUser,
  unique,
} from "../helpers";

test.use({
  viewport: { width: 375, height: 812 },
  hasTouch: true,
  isMobile: true,
});

test("creating a collection from the bookmark menu on mobile closes the menu", async ({
  page,
}) => {
  await registerUser(page, unique("coll"));
  const title = unique("Collected thread");
  await createThread(page, title, "body text");

  const category = await ensureThreadCategory();
  await page.goto("/");
  await page.getByRole("link", { name: category }).first().click();
  await page.waitForURL(/\/d\//);

  const bookmark = page
    .locator('button[data-scope="menu"][data-part="trigger"]')
    .filter({ has: page.locator("svg.lucide-bookmark") })
    .first();
  await bookmark.tap();

  const menu = page.getByRole("menu");
  await expect(menu).toBeVisible();
  await page.getByRole("menuitem", { name: "Collection" }).tap();

  const dialog = page.getByRole("dialog", { name: "Create collection" });
  await expect(dialog).toBeVisible();
  await expect(menu).toBeHidden();

  const name = unique("Sammlung");
  await dialog.getByRole("textbox").first().fill(name);
  await dialog.getByRole("button", { name: "Create", exact: true }).tap();
  await expect(dialog).toBeHidden();

  await bookmark.tap();
  await page.getByRole("menuitem", { name }).tap();
  await expect(menu).toBeHidden();
  await expect(bookmark).toBeVisible();

  await bookmark.tap();
  await expect(page.getByRole("menuitem", { name })).toBeVisible();
});
