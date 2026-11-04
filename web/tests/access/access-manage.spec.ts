import { expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import { PASSWORD, dismissOnboarding, registerUser, unique } from "../helpers";

test("managers confirm payment, extend and revoke access", async ({
  page,
  browser,
}) => {
  const seed = unique("mgr");
  const secret = `Protokollinhalt ${seed}`;

  const { category, thread, pass } = await withAdminAccessKey(
    async ({ categoryCreate, threadCreate, accessPassCreate }) => {
      const category = await categoryCreate({
        colour: "#3b82f6",
        description: `Locked ${seed}`,
        name: `Z2 ${seed}`,
        slug: `z2-${seed}`,
      });
      const thread = await threadCreate({
        title: `Protokoll Physiologie ${seed}`,
        body: `<p>${secret}</p>`,
        category: category.id,
        visibility: "published",
      });
      const pass = await accessPassCreate({
        name: `Z2-Protokolle ${seed}`,
        price_label: "10 €",
        category_ids: [category.id],
      });
      return { category, thread, pass };
    },
  );

  const memberContext = await browser.newContext();
  const member = await memberContext.newPage();
  const memberHandle = unique("student");
  await registerUser(member, memberHandle);
  await member.goto(`/t/${thread.slug}`);
  const card = member.getByRole("region", { name: pass.name });
  await card.getByRole("button", { name: "Request access" }).click();
  const reference = await card
    .getByTestId("access-reference-code")
    .textContent();
  expect(reference).toMatch(/^[A-Z0-9]{1,4}-[A-Z2-9]{4}$/);

  const adminHandle = unique("kassenwart");
  await createAdmin(page.context(), adminHandle, PASSWORD);
  await login(page, adminHandle, PASSWORD);
  await dismissOnboarding(page);

  await page.goto("/access?tab=requests");
  const request = page
    .getByTestId("access-grant-row")
    .filter({ hasText: `@${memberHandle}` });
  await expect(request).toBeVisible({ timeout: 10000 });
  await expect(request.getByTestId("access-grant-reference")).toHaveText(
    reference!,
  );

  await request.getByRole("button", { name: "Paid – unlock" }).click();
  const dialog = page.getByTestId("access-grant-dialog");
  await expect(
    dialog.getByRole("radio", { name: /^End of / }).first(),
  ).toHaveAttribute("aria-checked", "true");
  await dialog.getByRole("textbox", { name: /^Note/ }).fill("10 € bar");
  await dialog.getByRole("button", { name: "Unlock", exact: true }).click();

  await expect(request).toHaveCount(0, { timeout: 10000 });

  await member.reload();
  await expect(member.getByText(secret).first()).toBeVisible({
    timeout: 10000,
  });

  await page.getByRole("tab", { name: "Unlocked" }).click();
  await page.getByRole("textbox", { name: "Search by name" }).fill(memberHandle);
  const granted = page
    .getByTestId("access-grant-row")
    .filter({ hasText: `@${memberHandle}` });
  await expect(granted).toBeVisible({ timeout: 10000 });
  await expect(granted.getByText("10 € bar")).toBeVisible();
  await expect(granted.getByText(/^until /)).toBeVisible();

  await granted.getByRole("button", { name: "Extend" }).click();
  await dialog.getByRole("radio", { name: "No end date" }).click();
  await dialog.getByRole("button", { name: "Unlock", exact: true }).click();
  await expect(granted.getByText("no end date")).toBeVisible({
    timeout: 10000,
  });

  await granted.getByRole("button", { name: "Revoke" }).click();
  await granted.getByRole("button", { name: "Revoke this access?" }).click();
  await expect(granted).toHaveCount(0, { timeout: 10000 });

  await member.reload();
  await expect(member.getByTestId("access-locked-notice")).toBeVisible({
    timeout: 10000,
  });
  await expect(member.getByText(secret)).toHaveCount(0);

  await memberContext.close();
});

test("managers create a pass for a category", async ({ page }) => {
  const seed = unique("pass");

  const category = await withAdminAccessKey(({ categoryCreate }) =>
    categoryCreate({
      colour: "#3b82f6",
      description: `Pass target ${seed}`,
      name: `Z3 ${seed}`,
      slug: `z3-${seed}`,
    }),
  );

  const adminHandle = unique("kassenwart");
  await createAdmin(page.context(), adminHandle, PASSWORD);
  await login(page, adminHandle, PASSWORD);
  await dismissOnboarding(page);

  await page.goto("/access?tab=passes");
  await page.getByRole("button", { name: "Create pass" }).click();

  const editor = page.getByTestId("access-pass-editor");
  const name = `Z3-Protokolle ${seed}`;
  await editor.getByRole("textbox", { name: "Name" }).fill(name);
  await editor.getByRole("textbox", { name: "Price label" }).fill("10 €");
  await editor
    .getByRole("textbox", { name: "Payment instructions" })
    .fill("Bar im Fachschaftsraum");
  await editor.getByText(category.name, { exact: true }).click();
  await editor.getByRole("button", { name: "Save" }).click();

  const row = page.getByRole("group", { name }).or(
    page.locator(`[aria-label="${name}"]`),
  );
  await expect(row.first()).toBeVisible({ timeout: 10000 });
  await expect(row.first().getByText(category.name)).toBeVisible();

  await page.goto(`/d/${category.slug}`);
  await expect(page.getByRole("region", { name })).toHaveCount(0);
});
