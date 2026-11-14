import { Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { dismissOnboarding, registerUser, unique } from "../helpers";

async function expectListCard(
  page: Page,
  seed: string,
  thread: { body?: string; url?: string },
  url: string,
) {
  const title = `Link thread ${seed}`;

  const { slug, threadSlug } = await withAdminAccessKey(
    async ({ categoryCreate, threadCreate }) => {
      const category = await categoryCreate({
        colour: "#3b82f6",
        description: `Link preview ${seed}`,
        name: `Link Category ${seed}`,
        slug: `link-category-${seed}`,
      });

      const created = await threadCreate({
        title,
        body: thread.body ?? `<p>${title}</p>`,
        category: category.id,
        visibility: "published",
        url: thread.url,
        meta: { semester: { term: "2026-SS" } },
      });

      return { slug: category.slug, threadSlug: created.slug };
    },
  );

  await registerUser(page, unique("lnk-viewer"));
  await page.setViewportSize({ width: 375, height: 812 });
  await page.goto(`/d/${slug}`);
  await dismissOnboarding(page);

  const section = page.getByRole("region", { name: "SS26", exact: true });
  await expect(
    section.getByText(title).filter({ visible: true }).first(),
  ).toBeVisible({ timeout: 10000 });

  const card = section.locator(`a[href="${url}"]`).filter({ visible: true });
  await expect(card).toBeVisible();
  await expect(card).toContainText(new URL(url).hostname);

  return threadSlug;
}

test("shows the link preview card in the semester-grouped category list", async ({
  page,
}) => {
  const seed = unique("lnk");
  const url = `https://link-preview-${seed}.invalid/page`;
  await expectListCard(page, seed, { url }, url);
});

test("shows an embedded link card from the body in the category list", async ({
  page,
}) => {
  const seed = unique("card");
  const url = `https://link-card-${seed}.invalid/`;
  const threadSlug = await expectListCard(
    page,
    seed,
    {
      body: `<p>Altklausuren ${seed}</p><div data-href="${url}" data-display="card"></div>`,
    },
    url,
  );

  await page.goto(`/t/${threadSlug}`);
  const cards = page.locator(`a[href="${url}"]`).filter({ visible: true });
  await expect(cards.first()).toBeVisible({ timeout: 10000 });
  await expect(cards).toHaveCount(1);
});
