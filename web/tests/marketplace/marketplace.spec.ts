import { Page, expect, test } from "@playwright/test";

import { ONE_PIXEL_PNG, login, logout, registerUser, unique } from "../helpers";

async function setPhone(page: Page, phone: string) {
  await page.goto("/settings?tab=account");
  const input = page.locator('form#phone input[name="phone"]');
  await input.fill(phone);
  await page.locator("form#phone").getByRole("button", { name: "Save" }).click();
  await expect(page.getByText("Phone number saved")).toBeVisible({
    timeout: 10000,
  });
}

test.describe("Flea market", () => {
  test("listing, offer and WhatsApp contact", async ({ page }) => {
    const seller = unique("seller");
    const buyer = unique("buyer");
    const title = unique("Stethoskop Littmann");

    await registerUser(page, seller);

    await page.getByRole("link", { name: "Flea market" }).first().click();
    await expect(page).toHaveURL(/\/flohmarkt/);

    await page.getByRole("link", { name: "New listing" }).click();
    await expect(page).toHaveURL("/flohmarkt/new");
    await expect(page.getByTestId("phone-required")).toBeVisible();

    await setPhone(page, "0664 1112233");

    await page.goto("/flohmarkt/new");
    await page.getByTestId("listing-image-input").setInputFiles({
      name: "stethoskop.png",
      mimeType: "image/png",
      buffer: ONE_PIXEL_PNG,
    });
    await expect(page.getByTestId("listing-image")).toHaveCount(1, {
      timeout: 15000,
    });

    await page.locator('input[name="title"]').fill(title);
    await page.locator('select[name="price_type"]').selectOption("negotiable");
    await page.locator('input[name="price"]').fill("45");
    await page.locator('select[name="category"]').selectOption("instruments");
    await page.locator('input[name="location"]').fill("Wien");
    await page.locator('textarea[name="description"]').fill("Kaum benutzt.");
    await page.getByRole("button", { name: "Publish listing" }).click();

    await expect(page).toHaveURL(/\/flohmarkt\/[a-z0-9]+-/, { timeout: 15000 });
    const listingURL = new URL(page.url()).pathname;
    await expect(page.getByRole("heading", { name: title })).toBeVisible();
    await expect(page.getByTestId("listing-price")).toContainText("45");
    await expect(page.getByTestId("listing-price")).toContainText("VB");

    await logout(page);

    await registerUser(page, buyer);
    await setPhone(page, "+49 151 99887766");

    await page.goto("/flohmarkt");
    await page.getByRole("searchbox").fill(title);
    await page.getByRole("button", { name: "Search" }).click();
    const card = page.getByTestId("listing-card").filter({ hasText: title });
    await expect(card).toBeVisible({ timeout: 10000 });
    await card.getByRole("link", { name: title }).click();
    await expect(page).toHaveURL(listingURL);

    await expect(page.getByTestId("whatsapp-link")).toHaveCount(0);

    await page.locator('input[name="amount"]').fill("40");
    await page.locator('input[name="message"]').fill("Abholung am Freitag?");
    await page.getByRole("button", { name: "Send offer" }).click();

    const whatsapp = page
      .getByTestId("listing-contact")
      .getByTestId("whatsapp-link");
    await expect(whatsapp).toBeVisible({ timeout: 10000 });
    await expect(whatsapp).toHaveAttribute(
      "href",
      /^https:\/\/wa\.me\/436641112233\?text=/,
    );
    await expect(page.getByTestId("contact-phone")).toHaveText("+436641112233");

    await logout(page);

    await login(page, seller);
    await page.goto(listingURL);
    const offers = page.getByTestId("listing-offers");
    await expect(offers).toContainText("Abholung am Freitag?", {
      timeout: 10000,
    });
    await expect(offers.getByTestId("whatsapp-link")).toHaveAttribute(
      "href",
      /^https:\/\/wa\.me\/4915199887766\?text=/,
    );
  });
});
