import { Page, expect, test } from "@playwright/test";

const KONAMI = [
  "ArrowUp",
  "ArrowUp",
  "ArrowDown",
  "ArrowDown",
  "ArrowLeft",
  "ArrowRight",
  "ArrowLeft",
  "ArrowRight",
  "b",
  "a",
];

const UNLOCKED = /Konami/;

async function typeKeys(page: Page, keys: string[]) {
  for (const key of keys) {
    await page.keyboard.press(key);
  }
}

test.describe("Konami easter egg", () => {
  test("reveals the secret after entering the Konami code", async ({
    page,
  }) => {
    await page.goto("/");
    await page.locator("body").click({ position: { x: 1, y: 1 } });

    await typeKeys(page, KONAMI);

    await expect(page.getByText(UNLOCKED)).toBeVisible();
    await expect(page.getByTestId("konami-easter-egg")).toBeAttached();
  });

  test("does nothing for a wrong sequence", async ({ page }) => {
    await page.goto("/");
    await page.locator("body").click({ position: { x: 1, y: 1 } });

    await typeKeys(page, [...KONAMI.slice(0, 8), "a", "b"]);

    await expect(page.getByTestId("konami-easter-egg")).toHaveCount(0);
    await expect(page.getByText(UNLOCKED)).toHaveCount(0);
  });
});
