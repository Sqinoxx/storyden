import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { dismissOnboarding, registerUser, unique } from "../helpers";

test("shows the link preview card in the semester-grouped category list", async ({
  page,
}) => {
  const seed = unique("lnk");
  const title = `Link thread ${seed}`;
  const url = `https://link-preview-${seed}.invalid/page`;

  const slug = await withAdminAccessKey(
    async ({ categoryCreate, threadCreate }) => {
      const category = await categoryCreate({
        colour: "#3b82f6",
        description: `Link preview ${seed}`,
        name: `Link Category ${seed}`,
        slug: `link-category-${seed}`,
      });

      await threadCreate({
        title,
        body: `<p>${title}</p>`,
        category: category.id,
        visibility: "published",
        url,
        meta: { semester: { term: "2026-SS" } },
      });

      return category.slug;
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
  await expect(card).toContainText(`link-preview-${seed}.invalid`);
});
