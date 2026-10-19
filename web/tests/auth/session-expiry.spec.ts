import { expect, test } from "@playwright/test";

import { registerUser, unique } from "../helpers";

test.describe("Session expiry", () => {
  test("should redirect to login when a request is sent with an invalid session", async ({
    page,
    context,
  }) => {
    await registerUser(page, unique("expiry"));
    await page.goto("/settings");

    const form = page.locator("form").filter({
      has: page.getByRole("heading", { name: /Interface settings|Oberfläche/ }),
    });
    const save = form.getByRole("button", { name: /^(Save|Speichern)$/ });
    await expect(save).toBeVisible({ timeout: 10000 });

    await context.clearCookies();
    await save.click();

    await expect(page).toHaveURL("/login", { timeout: 10000 });
    await expect(page.getByRole("button", { name: "Log In" })).toBeVisible();
    await expect(
      page.getByText("The request did not contain any authentication"),
    ).toHaveCount(0);
  });
});
