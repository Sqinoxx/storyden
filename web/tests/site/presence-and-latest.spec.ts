import { Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { dismissOnboarding, registerUser, unique } from "../helpers";

async function seedCategory(seed: string) {
  const slug = `presence-${seed}`.toLowerCase();
  await withAdminAccessKey(async ({ categoryCreate }) => {
    await categoryCreate({
      colour: "#3b82f6",
      description: `Presence ${seed}`,
      name: `Presence ${seed}`,
      slug,
    });
  });
  return slug;
}

async function expectNoHorizontalOverflow(page: Page) {
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(overflow).toBe(false);
}

test.describe("Presence indicator", () => {
  test("shows online members and others studying the same category", async ({
    page,
    browser,
  }) => {
    const slug = await seedCategory(unique("pr"));

    await registerUser(page, unique("presence-a"));
    await dismissOnboarding(page);

    const indicator = page.getByTestId("presence-indicator").filter({ visible: true });
    await expect(indicator).toBeVisible();
    await expect(indicator).toContainText("online");

    const other = await browser.newContext();
    const otherPage = await other.newPage();
    await registerUser(otherPage, unique("presence-b"));
    await otherPage.goto(`/d/${slug}`);
    await expect(
      otherPage.getByTestId("presence-indicator").filter({ visible: true }),
    ).toBeVisible();

    await page.goto(`/d/${slug}`);
    await expect(
      page.getByTestId("presence-in-category").filter({ visible: true }),
    ).toHaveText("1 here");

    await other.close();
  });

  test("is hidden for guests", async ({ page }) => {
    await page.goto("/");
    await expect(page.getByTestId("presence-indicator")).toHaveCount(0);
  });

  test("fits the mobile navigation bar", async ({ page }) => {
    await page.setViewportSize({ width: 375, height: 812 });
    await registerUser(page, unique("presence-m"));
    await dismissOnboarding(page);

    await expect(
      page.getByTestId("presence-indicator").filter({ visible: true }),
    ).toBeVisible();
    await expectNoHorizontalOverflow(page);
  });
});

test.describe("Latest posts tab", () => {
  test("members reach latest posts from the sidebar", async ({
    page,
  }) => {
    await registerUser(page, unique("latest-d"));
    await dismissOnboarding(page);

    await page.getByRole("button", { name: "Open navigation sidebar" }).click();
    await page
      .getByRole("navigation")
      .getByRole("link", { name: "Latest posts", exact: true })
      .first()
      .click();
    await expect(page).toHaveURL("/d");
  });

  test("members reach latest posts from the mobile menu", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await registerUser(page, unique("latest-m"));
    await dismissOnboarding(page);

    await page.getByRole("button", { name: "Main navigation menu" }).click();
    await page
      .getByRole("link", { name: "Latest posts", exact: true })
      .first()
      .click();
    await expect(page).toHaveURL("/d");
    await expectNoHorizontalOverflow(page);
  });
});
