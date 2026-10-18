import { expect, test } from "@playwright/test";

test.describe("Logout", () => {
  test("should logout successfully", async ({ page }) => {
    const timestamp = Date.now();
    const username = `testuser${timestamp}`;
    const password = "TestPassword123!";

    await page.goto("/register");
    await page.getByRole("textbox", { name: "username" }).fill(username);
    await page
      .getByRole("combobox", { name: "Fachsemester" })
      .selectOption({ label: "1. Fachsemester" });
    await page.getByRole("textbox", { name: "password" }).fill(password);
    await page.getByRole("button", { name: "Register" }).click();

    await expect(page).not.toHaveURL(/\/register/, { timeout: 10000 });

    await page.getByRole("button", { name: "Account menu" }).click();
    await page.getByRole("menuitem", { name: "Logout" }).click();

    await expect(page).toHaveURL("/login", { timeout: 10000 });
    await expect(page.getByRole("button", { name: "Log In" })).toBeVisible();
    await expect(page.getByText("An unexpected error occurred")).toHaveCount(0);
  });

  test("should redirect to login when the session has expired", async ({
    page,
    context,
    baseURL,
  }) => {
    await context.addCookies([
      { name: "storyden-session", value: "expired", url: baseURL! },
    ]);

    await page.goto("/");

    await expect(page).toHaveURL("/login", { timeout: 10000 });
    await expect(page.getByRole("button", { name: "Log In" })).toBeVisible();
    await expect(page.getByText("Fehler")).toHaveCount(0);
  });
});
