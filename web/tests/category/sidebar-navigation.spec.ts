import { Page, expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import { PASSWORD, dismissOnboarding, openSidebar, unique } from "../helpers";

function sidebarRow(page: Page, name: string) {
  return page.locator('[data-part="branch-control"]').filter({
    has: page.locator('[data-part="branch-text"]', {
      hasText: new RegExp(`^${name}$`),
    }),
  });
}

test("sidebar category rows navigate on a single click and the chevron only toggles", async ({
  page,
}) => {
  const seed = unique("sidenav");
  const parentName = `Parent ${seed}`;
  const childName = `Child ${seed}`;
  const parentSlug = `sidenav-parent-${seed}`;
  const childSlug = `sidenav-child-${seed}`;

  await withAdminAccessKey(async ({ categoryCreate }) => {
    const parent = await categoryCreate({
      colour: "#3b82f6",
      description: `Sidebar parent ${seed}`,
      name: parentName,
      slug: parentSlug,
    });
    await categoryCreate({
      colour: "#3b82f6",
      description: `Sidebar child ${seed}`,
      name: childName,
      slug: childSlug,
      parent: parent.id,
    });
  });

  const admin = unique("sidenavadmin").replace(/-/g, "");
  await createAdmin(page.context(), admin, PASSWORD);
  await login(page, admin, PASSWORD);
  await page.goto("/d");
  await dismissOnboarding(page);
  await openSidebar(page);

  const parentRow = sidebarRow(page, parentName);
  const childRow = sidebarRow(page, childName);
  await expect(childRow).toBeVisible();

  await parentRow.getByRole("button", { name: "Collapse section" }).click();
  await expect(childRow).toBeHidden();
  await expect(page).toHaveURL(/\/d$/);

  await parentRow.getByRole("button", { name: "Expand section" }).click();
  await expect(childRow).toBeVisible();
  await expect(page).toHaveURL(/\/d$/);

  await childRow.click({ position: { x: 4, y: 8 } });
  await expect(page).toHaveURL(new RegExp(`/d/${childSlug}$`));

  await parentRow.getByText(parentName, { exact: true }).click();
  await expect(page).toHaveURL(new RegExp(`/d/${parentSlug}$`));
  await expect(childRow).toBeVisible();
});
