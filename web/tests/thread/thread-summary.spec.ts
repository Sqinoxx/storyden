import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { createThread, postReply, registerUser, unique } from "../helpers";

test.describe("Thread summary", () => {
  test.beforeAll(async () => {
    await withAdminAccessKey(async ({ adminSettingsUpdate }) => {
      await adminSettingsUpdate({
        services: { ai: { thread_summary: true } },
      });
    });
  });

  test("streams and keeps the summary of a long thread", async ({ page }) => {
    await registerUser(page, unique("threadsum"));
    await createThread(
      page,
      unique("Hirnnerven Eselsbruecke"),
      "Kennt jemand eine Eselsbruecke fuer die Hirnnerven?",
    );

    for (let i = 1; i <= 5; i++) {
      await postReply(page, unique(`Antwort ${i} zu den Hirnnerven`));
    }

    await page.reload();

    const button = page.getByRole("button", { name: "Summarise" });
    await expect(button).toBeVisible({ timeout: 15000 });
    await button.click();

    const summary = page.getByTestId("thread-summary");
    await expect(summary).toBeVisible({ timeout: 30000 });
    await expect(summary).toContainText("An answer for");
    await expect(summary).toContainText("Eselsbruecke");

    await page.reload();
    await page.getByRole("button", { name: "Summarise" }).click();
    await expect(page.getByTestId("thread-summary")).toContainText(
      "An answer for",
    );
  });
});
