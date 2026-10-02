import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { registerUser, unique } from "../helpers";

async function seedTaggedThread(seed: string) {
  const title = `Tagged thread ${seed}`;
  const tag = `topic-${seed}`.toLowerCase();

  await withAdminAccessKey(async ({ categoryCreate, threadCreate }) => {
    const category = await categoryCreate({
      colour: "#3b82f6",
      description: `Tags ${seed}`,
      name: `Tags Category ${seed}`,
      slug: `tags-category-${seed}`,
    });

    await threadCreate({
      title,
      body: `<p>${title}</p>`,
      category: category.id,
      visibility: "published",
      tags: [tag],
    });
  });

  return { title, tag };
}

test.describe("Tags index", () => {
  test("members can browse tags and open tagged threads", async ({ page }) => {
    const { title, tag } = await seedTaggedThread(unique("tg"));
    await registerUser(page, unique("tag-reader"));

    await page.getByRole("button", { name: "Open navigation sidebar" }).click();
    const nav = page.getByRole("navigation");
    await expect(
      nav.getByRole("link", { name: "Tag Management" }),
    ).toHaveCount(0);
    await nav.getByRole("link", { name: "Tags", exact: true }).first().click();
    await expect(page).toHaveURL("/tags");

    await expect(
      page.getByRole("button", { name: "Create Tag" }),
    ).toHaveCount(0);

    await page.getByPlaceholder("Search tags...").fill(tag);
    await page.getByRole("link", { name: tag }).click();
    await expect(page).toHaveURL(`/tags/${tag}`);

    await expect(page.getByRole("heading", { name: "Threads" })).toBeVisible();
    const threadLink = page
      .getByRole("heading", { name: title })
      .getByRole("link", { name: title });
    await threadLink.focus();
    await page.keyboard.press("Enter");
    await expect(page).toHaveURL(/\/t\//);
    await expect(page.getByRole("heading", { name: title })).toBeVisible();
  });

  test("members can search, filter and sort a tag's items", async ({
    page,
  }) => {
    const seed = unique("tf");
    const tag = `filter-${seed}`.toLowerCase();
    const alpha = `Alpha ${seed}`;
    const beta = `Beta ${seed}`;
    const betaCategory = `Beta Category ${seed}`;

    await withAdminAccessKey(async ({ categoryCreate, threadCreate }) => {
      const first = await categoryCreate({
        colour: "#3b82f6",
        description: `Alpha ${seed}`,
        name: `Alpha Category ${seed}`,
        slug: `alpha-category-${seed}`,
      });
      const second = await categoryCreate({
        colour: "#22c55e",
        description: `Beta ${seed}`,
        name: betaCategory,
        slug: `beta-category-${seed}`,
      });

      for (const [title, category] of [
        [alpha, first.id],
        [beta, second.id],
      ]) {
        await threadCreate({
          title,
          body: `<p>${title}</p>`,
          category,
          visibility: "published",
          tags: [tag],
        });
      }
    });

    await registerUser(page, unique("tag-filter"));
    await page.goto(`/tags/${tag}`);

    const titles = page.getByRole("heading", { name: seed });
    const alphaHeading = page.getByRole("heading", { name: alpha });
    const betaHeading = page.getByRole("heading", { name: beta });

    await expect(betaHeading).toBeVisible();
    await expect(titles.first()).toHaveText(beta);

    await page.getByRole("combobox", { name: "Sort by" }).click();
    await page.getByRole("option", { name: "A–Z" }).click();
    await expect(titles.first()).toHaveText(alpha);
    await expect(page).toHaveURL(/sort=alphabetical/);

    await page
      .getByPlaceholder("Search threads and pages...")
      .fill("beta");
    await expect(betaHeading).toBeVisible();
    await expect(alphaHeading).toHaveCount(0);
    await expect(page.getByText("1 of 2 items")).toBeVisible();
    await expect(page).toHaveURL(/q=beta/);

    await page.reload();
    await expect(
      page.getByPlaceholder("Search threads and pages..."),
    ).toHaveValue("beta");
    await expect(alphaHeading).toHaveCount(0);

    await page.getByRole("button", { name: "Reset filters" }).click();
    await expect(alphaHeading).toBeVisible();

    await page.getByRole("combobox", { name: "Category" }).click();
    await page.getByRole("option", { name: betaCategory }).click();
    await expect(betaHeading).toBeVisible();
    await expect(alphaHeading).toHaveCount(0);

    await page.getByRole("combobox", { name: "Content type" }).click();
    await page.getByRole("option", { name: "Library pages" }).click();
    await expect(
      page.getByText("No items match your filters."),
    ).toBeVisible();
  });

  test("members reach tags from the mobile menu", async ({ page }) => {
    await page.setViewportSize({ width: 390, height: 844 });
    await registerUser(page, unique("tag-mobile"));

    await page.getByRole("button", { name: "Main navigation menu" }).click();
    await page.getByRole("link", { name: "Tags", exact: true }).first().click();
    await expect(page).toHaveURL("/tags");
    await expect(page.getByPlaceholder("Search tags...")).toBeVisible();

    const overflow = await page.evaluate(
      () => document.documentElement.scrollWidth > window.innerWidth,
    );
    expect(overflow).toBe(false);
  });

  test("members cannot open tag management", async ({ page }) => {
    await registerUser(page, unique("tag-member"));

    await page.goto("/admin/tags");
    await expect(
      page.getByText("Nur für Moderatoren und Admins sichtbar."),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Create Tag" }),
    ).toHaveCount(0);
  });

  test("guests are asked to log in", async ({ page }) => {
    await page.goto("/tags");
    await expect(page.getByText("Authentication Required")).toBeVisible();
  });
});
