import { expect, test } from "@playwright/test";

import {
  createAdmin,
  login,
  withAdminAccessKey,
} from "../access_key_admin_assignment";
import { PASSWORD, dismissOnboarding, unique } from "../helpers";

const LABEL = "Library assistance";

test.describe("Admin AI settings", () => {
  test.afterAll(async () => {
    await withAdminAccessKey(async ({ adminSettingsUpdate }) => {
      await adminSettingsUpdate({ services: { ai: { library_assist: true } } });
    });
  });

  test("switches toggle and persist", async ({ page, context }) => {
    await withAdminAccessKey(async ({ adminSettingsUpdate }) => {
      await adminSettingsUpdate({ services: { ai: { library_assist: true } } });
    });

    const admin = unique("aiadmin");
    await createAdmin(context, admin, PASSWORD);
    await login(page, admin, PASSWORD);

    await page.goto("/admin?tab=ai");
    await dismissOnboarding(page);

    const input = page.getByLabel(LABEL);
    await expect(input).toBeChecked();

    await page.getByText(LABEL, { exact: true }).click();
    await expect(input).not.toBeChecked();

    await page.getByRole("button", { name: "Save" }).click();
    await expect(page.getByText("Settings saved")).toBeVisible();

    await page.reload();
    await expect(page.getByLabel(LABEL)).not.toBeChecked();
  });
});
