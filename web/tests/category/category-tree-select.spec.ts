import { Locator, Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import {
  dismissOnboarding,
  quickShareForm,
  registerUser,
  unique,
} from "../helpers";

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
  // Scoped to the popover's own tree panel: the persistent sidebar navigation
  // renders a tree of the same categories with the same [data-part] markup,
  // so an unscoped lookup is ambiguous whenever a category name appears in
  // both places.
  return page
    .getByTestId("category-tree-panel")
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
  // getByText(title) is ambiguous here: Next.js's route announcer mirrors the
  // page title into a live region with the same text.
  await expect(page.getByRole("heading", { name: title })).toBeVisible({
    timeout: 10000,
  });
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

test("a user allowed to post in any category can QuickShare into the category being viewed, not just its subcategory", async ({
  page,
}) => {
  const seed = unique("qsanycat");
  const { names } = await seedCategoryTree(seed);

  // Sign up through the browser (known-good flow) and grant admin afterwards,
  // rather than the access-key-assignment package's own login() - its account
  // page markup has since drifted and no longer exposes a "username" textbox.
  const username = unique("qsadmin").replace(/-/g, "");
  await registerUser(page, username);
  await withAdminAccessKey(async ({ accountAddRole }) => {
    await accountAddRole(username, "00000000000000000a00");
  });

  // The branch category has a subcategory (the leaf), so this is exactly the
  // shape that used to force even a privileged poster down into the leaf.
  await page.goto(`/d/tree-branch-${seed}`);
  await dismissOnboarding(page);

  const form = quickShareForm(page);
  await expect(form).toBeVisible();

  await form.getByPlaceholder("Thread title...").fill(`QuickShare into branch ${seed}`);
  await form.locator(".ProseMirror").first().fill("posted straight into the viewed category");

  await categoryPicker(page).click();

  // The category being viewed is itself selectable directly - no need to
  // drill into its subcategory to find something postable. Selecting it
  // closes the popover, so its own trigger is the reliable place to confirm
  // what got picked (the tree panel behind it may still be mid-unmount).
  await categoryNode(page, names.branch).click();
  await expect(page.getByTestId("category-tree-panel")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: names.branch, exact: true }),
  ).toBeVisible();

  await form.getByRole("button", { name: "Share" }).click();

  await expect(
    page.getByText(`QuickShare into branch ${seed}`),
  ).toBeVisible({ timeout: 10000 });
});
