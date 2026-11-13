import { Page, expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import { PASSWORD, dismissOnboarding, openSidebar, unique } from "../helpers";

function sidebarBranch(page: Page, name: string) {
  return page
    .locator('[data-part="branch"]')
    .filter({
      has: page.locator('[data-part="branch-text"]', {
        hasText: new RegExp(`^${name}$`),
      }),
    })
    .last();
}

function sidebarRow(page: Page, name: string) {
  return page
    .locator('[data-part="branch-control"]')
    .filter({
      has: page.locator('[data-part="branch-text"]', {
        hasText: new RegExp(`^${name}$`),
      }),
    });
}

test("admin moves a category into another category from the sidebar menu", async ({
  page,
}) => {
  const seed = unique("moveinto");
  const target = `Target ${seed}`;
  const moved = `Moved ${seed}`;

  await withAdminAccessKey(async ({ categoryCreate }) => {
    await categoryCreate({
      colour: "#3b82f6",
      description: `Move into target ${seed}`,
      name: target,
      slug: `move-into-target-${seed}`,
    });
    await categoryCreate({
      colour: "#3b82f6",
      description: `Move into moved ${seed}`,
      name: moved,
      slug: `move-into-moved-${seed}`,
    });
  });

  const admin = unique("moveadmin").replace(/-/g, "");
  await createAdmin(page.context(), admin, PASSWORD);
  await login(page, admin, PASSWORD);
  await page.goto("/d");
  await dismissOnboarding(page);
  await openSidebar(page);

  await expect(
    page.locator('[data-part="trigger-item"]', {
      hasText: "Move into category",
    }),
  ).toHaveCount(0);

  await expect(
    sidebarBranch(page, target).locator('[data-part="branch-text"]', {
      hasText: new RegExp(`^${moved}$`),
    }),
  ).toHaveCount(0);

  const row = sidebarRow(page, moved);
  await row.hover();
  await row.getByRole("button", { name: "More options" }).click();

  await page.getByRole("menuitem", { name: "Move into category" }).click();
  await page
    .getByRole("menuitem")
    .filter({ hasText: new RegExp(`^${target}$`) })
    .click();

  await expect(
    sidebarBranch(page, target).locator('[data-part="branch-text"]', {
      hasText: new RegExp(`^${moved}$`),
    }),
  ).toHaveCount(1, { timeout: 10000 });
});
