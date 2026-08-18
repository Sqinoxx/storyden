import { Locator, Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { dismissOnboarding, registerUser, unique } from "../helpers";

type Seeded = {
  names: {
    root: string;
    branch: string;
    leaf: string;
  };
};

async function seedCategoryTree(seed: string): Promise<Seeded> {
  const names = {
    root: `Vorklinik ${seed}`,
    branch: `3. Semester ${seed}`,
    leaf: `PPZ ${seed}`,
  };

  await withAdminAccessKey(async ({ categoryCreate }) => {
    const root = await categoryCreate({
      colour: "#3b82f6",
      description: `Tree select root ${seed}`,
      name: names.root,
      slug: `tree-root-${seed}`,
    });

    const branch = await categoryCreate({
      colour: "#3b82f6",
      description: `Tree select branch ${seed}`,
      name: names.branch,
      slug: `tree-branch-${seed}`,
      parent: root.id,
    });

    await categoryCreate({
      colour: "#3b82f6",
      description: `Tree select leaf ${seed}`,
      name: names.leaf,
      slug: `tree-leaf-${seed}`,
      parent: branch.id,
    });
  });

  return { names };
}

function categoryPicker(page: Page) {
  return page.getByRole("button", { name: "Select category" });
}

function categoryNode(page: Page, name: string): Locator {
  return page
    .locator('[data-part="branch-text"]')
    .filter({ hasText: new RegExp(`^${name}$`) });
}

async function openComposer(page: Page, title: string) {
  await page.goto("/new");
  await dismissOnboarding(page);

  await page.getByPlaceholder("Thread title...").fill(title);
  await page.locator(".ProseMirror").first().fill("a tree select post");
}

test("drills through the category tree to reach a leaf", async ({ page }) => {
  const seed = unique("tree");
  const { names } = await seedCategoryTree(seed);
  const title = `Tree select post ${seed}`;

  await registerUser(page, unique("treeuser").replace(/-/g, ""));
  await openComposer(page, title);

  await categoryPicker(page).click();

  // Only the roots are listed until a branch is expanded.
  await expect(categoryNode(page, names.root)).toBeVisible();
  await expect(categoryNode(page, names.branch)).toBeHidden();

  // A category with sub-categories expands instead of being selected.
  await categoryNode(page, names.root).click();
  await expect(categoryNode(page, names.branch)).toBeVisible();
  await expect(categoryPicker(page)).toBeVisible();

  await categoryNode(page, names.branch).click();
  await expect(categoryNode(page, names.leaf)).toBeVisible();

  await categoryNode(page, names.leaf).click();

  const selected = page.getByRole("button", {
    name: `${names.root} / ${names.branch} / ${names.leaf}`,
  });
  await expect(selected).toBeVisible();
  await expect(categoryPicker(page)).toHaveCount(0);

  await page.getByRole("button", { name: "Post", exact: true }).click();

  await expect(page).toHaveURL(/\/t\//, { timeout: 10000 });
  await expect(page.getByText(title)).toBeVisible({ timeout: 10000 });
});

test("members cannot select a category that has sub-categories", async ({
  page,
}) => {
  const seed = unique("leafonly");
  const { names } = await seedCategoryTree(seed);

  await registerUser(page, unique("leafuser").replace(/-/g, ""));
  await openComposer(page, `Leaf only ${seed}`);

  await categoryPicker(page).click();

  await categoryNode(page, names.root).click();
  await categoryNode(page, names.branch).click();

  // Both ancestors stayed unselected — the picker still shows its placeholder.
  await expect(categoryPicker(page)).toBeVisible();
  await expect(page.getByRole("button", { name: "No category" })).toHaveCount(
    0,
  );
});

test("members without POST_IN_ANY_CATEGORY cannot QuickShare on the uncategorised feed", async ({
  page,
}) => {
  const seed = unique("uncatqs");
  await seedCategoryTree(seed);

  await registerUser(page, unique("uncatuser").replace(/-/g, ""));

  await page.goto("/d");
  await dismissOnboarding(page);

  await expect(
    page.getByPlaceholder("Share a past exam, an exam question, or a tip..."),
  ).toHaveCount(0);
});

test("the category tree is usable at mobile widths", async ({ page }) => {
  const seed = unique("treemob");
  const { names } = await seedCategoryTree(seed);

  await page.setViewportSize({ width: 375, height: 812 });

  await registerUser(page, unique("mobuser").replace(/-/g, ""));
  await openComposer(page, `Mobile tree post ${seed}`);

  await categoryPicker(page).click();

  await categoryNode(page, names.root).click();
  await categoryNode(page, names.branch).click();

  const leaf = categoryNode(page, names.leaf);
  await expect(leaf).toBeInViewport();
  await leaf.click();

  await expect(
    page.getByRole("button", {
      name: `${names.root} / ${names.branch} / ${names.leaf}`,
    }),
  ).toBeVisible();
});
