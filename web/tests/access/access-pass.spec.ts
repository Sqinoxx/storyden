import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { registerUser, unique } from "../helpers";

async function seedLockedCategory(seed: string) {
  const secret = `Protokollinhalt ${seed}`;

  return withAdminAccessKey(
    async ({ categoryCreate, threadCreate, accessPassCreate }) => {
      const category = await categoryCreate({
        colour: "#3b82f6",
        description: `Locked ${seed}`,
        name: `Z1 ${seed}`,
        slug: `z1-${seed}`,
      });

      const thread = await threadCreate({
        title: `Protokoll Anatomie ${seed}`,
        body: `<p>${secret}</p>`,
        category: category.id,
        visibility: "published",
      });

      const pass = await accessPassCreate({
        name: `Z1-Protokolle ${seed}`,
        price_label: "10 €",
        payment_info: "Bar im Fachschaftsraum",
        category_ids: [category.id],
      });

      return { category, thread, pass, secret };
    },
  );
}

test("members request access, get unlocked and see the content", async ({
  page,
}) => {
  const seed = unique("acc");
  const { category, thread, pass, secret } = await seedLockedCategory(seed);

  const handle = unique("member");
  await registerUser(page, handle);

  await page.goto(`/d/${category.slug}`);

  await page.getByRole("button", { name: "Open navigation sidebar" }).click();
  const nav = page.getByRole("navigation");
  await expect(
    nav
      .getByRole("link", { name: new RegExp(`^${category.name}`) })
      .getByTestId("category-locked-icon"),
  ).toBeVisible({ timeout: 10000 });
  await expect(
    nav.getByRole("link", { name: "Access", exact: true }),
  ).toBeVisible();

  const card = page.getByRole("region", { name: pass.name });
  await expect(card).toBeVisible();
  await expect(card.getByText("Bar im Fachschaftsraum")).toBeVisible();

  await page.goto(`/t/${thread.slug}`);
  await expect(page.getByRole("heading", { name: thread.title })).toBeVisible();
  await expect(page.getByText(secret)).toHaveCount(0);

  const threadCard = page.getByRole("region", { name: pass.name });
  await threadCard.getByRole("button", { name: "Request access" }).click();

  await expect(threadCard.getByTestId("access-reference-code")).toHaveText(
    /^[A-Z0-9]{1,4}-[A-Z2-9]{4}$/,
  );
  await expect(
    threadCard.getByRole("button", { name: "Withdraw request" }),
  ).toBeVisible();

  await withAdminAccessKey(({ accessGrantSet }) =>
    accessGrantSet(pass.id, handle, {}),
  );

  await page.reload();
  await expect(page.getByText(secret).first()).toBeVisible({ timeout: 10000 });
  await expect(page.getByTestId("access-locked-notice")).toHaveCount(0);

  await page.goto("/access");
  const overview = page.getByRole("region", { name: pass.name });
  await expect(overview.getByText("Active")).toBeVisible();
  await expect(overview.getByText("Unlocked")).toBeVisible();
});

test("guests are asked to log in", async ({ page }) => {
  const seed = unique("acg");
  const { thread, pass, secret } = await seedLockedCategory(seed);

  await page.goto(`/t/${thread.slug}`);

  const card = page.getByRole("region", { name: pass.name });
  await expect(card).toBeVisible({ timeout: 10000 });
  await expect(
    card.getByRole("link", { name: "Log in to request access" }),
  ).toBeVisible();
  await expect(page.getByText(secret)).toHaveCount(0);
});
