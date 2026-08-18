import { Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { dismissOnboarding, registerUser, unique } from "../helpers";

const SS26 = "SS26";
const WS23 = "WS23/24";

async function seedCategory(seed: string) {
  const titles = {
    summer: `Summer exam ${seed}`,
    winter: `Winter exam ${seed}`,
  };

  const slug = await withAdminAccessKey(
    async ({ categoryCreate, threadCreate }) => {
      const category = await categoryCreate({
        colour: "#3b82f6",
        description: `Semester grouping ${seed}`,
        name: `Semester Category ${seed}`,
        slug: `semester-category-${seed}`,
      });

      for (const [title, term] of [
        [titles.summer, "2026-SS"],
        [titles.winter, "2023-WS"],
      ]) {
        await threadCreate({
          title: title!,
          body: `<p>${title}</p>`,
          category: category.id,
          visibility: "published",
          meta: { semester: { term } },
        });
      }

      return category.slug;
    },
  );

  return { slug, titles };
}

async function selectFeedView(page: Page, name: string) {
  await page.getByRole("button", { name: "Sort by" }).click();
  await page.getByRole("menuitem", { name, exact: true }).click();
}

function semesterSection(page: Page, label: string) {
  return page.getByRole("region", { name: label, exact: true });
}

test("groups category threads by semester", async ({ page }) => {
  const { slug, titles } = await seedCategory(unique("grp"));

  await page.goto(`/d/${slug}`);
  await dismissOnboarding(page);

  await expect(page.getByText(titles.summer)).toBeVisible({ timeout: 10000 });

  await selectFeedView(page, "By semester");

  await expect(page).toHaveURL(/sort=semester/);

  const summer = semesterSection(page, SS26);
  const winter = semesterSection(page, WS23);

  await expect(summer.getByText(titles.summer)).toBeVisible({ timeout: 10000 });
  await expect(winter.getByText(titles.winter)).toBeVisible();

  // The stored term wins over the creation date, otherwise both threads would
  // land in whichever term the suite happens to run in.
  await expect(summer.getByText(titles.winter)).toHaveCount(0);
  await expect(winter.getByText(titles.summer)).toHaveCount(0);
});

test("hides the semester view on mobile widths", async ({ page }) => {
  const { slug, titles } = await seedCategory(unique("mob"));

  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto(`/d/${slug}`);
  await dismissOnboarding(page);

  await expect(page.getByText(titles.summer)).toBeVisible({ timeout: 10000 });

  await page.getByRole("button", { name: "Sort by" }).click();

  await expect(
    page.getByRole("menuitem", { name: "Newest first", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByRole("menuitem", { name: "By semester", exact: true }),
  ).toBeHidden();
});

test("posting with an overridden semester files the thread under it", async ({
  page,
}) => {
  const seed = unique("post");
  const { slug } = await seedCategory(seed);
  const title = `Backdated exam ${seed}`;

  await registerUser(page, unique("semuser").replace(/-/g, ""));

  await page.goto(`/d/${slug}`);
  await dismissOnboarding(page);

  const composer = page
    .locator("form")
    .filter({ has: page.getByRole("button", { name: "Share" }) })
    .first();

  await composer.getByRole("textbox", { name: "Thread title" }).fill(title);
  await composer.locator(".ProseMirror").first().fill("an old exam");

  // The category page hides the category picker, so the composer's only select
  // is the semester one, which starts on the term containing today.
  const semester = composer.getByRole("combobox").first();
  await expect(semester).toBeVisible();
  await semester.click();
  await page.getByRole("option", { name: WS23, exact: true }).click();
  await expect(semester).toContainText(WS23);

  const post = composer.getByRole("button", { name: "Share" });
  await expect(post).toBeEnabled({ timeout: 10000 });
  await post.click();

  await expect(page.getByText(title)).toBeVisible({ timeout: 10000 });

  await selectFeedView(page, "By semester");

  await expect(semesterSection(page, WS23).getByText(title)).toBeVisible({
    timeout: 10000,
  });
});
