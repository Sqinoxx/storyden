import { expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";

const PASSWORD = "TestPassword123!";

test("admin statistics dashboard renders all sections", async ({
  page,
}, testInfo) => {
  const seed = Date.now().toString(36);
  const adminHandle = `stats-admin-${seed}`;
  const tagName = `stats-tag-${seed}`;

  await withAdminAccessKey(async ({ threadCreate }) => {
    await threadCreate({
      title: `Statistics thread ${seed}`,
      body: `Statistics body ${seed}`,
      visibility: "published",
      tags: [tagName],
    });
  });

  await createAdmin(page.context(), adminHandle, PASSWORD);
  await login(page, adminHandle, PASSWORD);

  await page.goto("/admin");
  await page.getByRole("tab", { name: "Statistiken" }).click();
  await expect(page).toHaveURL(/tab=statistics/);

  await expect(
    page.getByRole("heading", { name: "Nutzungsstatistiken" }),
  ).toBeVisible();

  for (const section of [
    "Überblick",
    "Wachstum",
    "Engagement",
    "Aktivitätsmuster",
    "Studium",
    "Inhalte & Moderation",
  ]) {
    await expect(
      page.getByRole("heading", { name: section, exact: true }),
    ).toBeVisible();
  }

  await expect(page.getByText("Neue Themen", { exact: true })).toBeVisible();
  await expect(page.getByText("Lade Statistiken…")).toHaveCount(0);
  await expect(page.getByText(/^Spitzenzeit:/)).toBeVisible();

  const tagsCard = page
    .getByRole("heading", { name: "Beliebteste Tags" })
    .locator("xpath=ancestor::div[3]");
  await expect(tagsCard.getByText(tagName)).toBeVisible();

  await page.screenshot({
    path: testInfo.outputPath("statistics-desktop.png"),
    fullPage: true,
  });

  await page.setViewportSize({ width: 390, height: 844 });
  await expect(
    page.getByRole("heading", { name: "Engagement", exact: true }),
  ).toBeVisible();
  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth > window.innerWidth,
  );
  expect(overflow).toBe(false);

  await page.screenshot({
    path: testInfo.outputPath("statistics-mobile.png"),
    fullPage: true,
  });
});
