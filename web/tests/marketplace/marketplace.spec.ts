import { Page, expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import { ONE_PIXEL_PNG, registerUser, unique } from "../helpers";

async function setMarketplaceEnabled(enabled: boolean) {
  await withAdminAccessKey(async ({ adminSettingsUpdate }) => {
    await adminSettingsUpdate({ services: { market: { enabled } } });
  });
}

// A fresh instance shows the onboarding checklist over the page content and it
// appears only once the client has loaded, so wait for that before dismissing.
async function clearOverlays(page: Page) {
  await page.waitForLoadState("networkidle");
  for (const name of [
    "Skip",
    "Hide this message",
    "Close navigation sidebar",
  ]) {
    const button = page.getByRole("button", { name });
    if (await button.isVisible({ timeout: 1500 }).catch(() => false)) {
      await button.click();
    }
  }
}

async function setPhone(page: Page, phone: string) {
  await page.goto("/settings?tab=account");
  await clearOverlays(page);
  const input = page.locator('form#phone input[name="phone"]');
  await input.fill(phone);
  await page
    .locator("form#phone")
    .getByRole("button", { name: "Save" })
    .click();
  await expect(page.getByText("Phone number saved")).toBeVisible({
    timeout: 10000,
  });
}

test.describe("Flea market", () => {
  test.describe.configure({ mode: "serial" });

  test("hidden from members while in beta", async ({ page }) => {
    await setMarketplaceEnabled(false);

    await registerUser(page, unique("member"));
    await page.getByRole("button", { name: "Open navigation sidebar" }).click();
    await expect(
      page.getByRole("link", { name: "Latest posts" }),
    ).toBeVisible();
    await expect(page.getByRole("link", { name: /Flea market/ })).toHaveCount(
      0,
    );

    const response = await page.goto("/flohmarkt");
    expect(response?.status()).toBe(404);
  });

  test("listing, offer and WhatsApp contact", async ({ browser }) => {
    await setMarketplaceEnabled(true);

    const sellerContext = await browser.newContext();
    const buyerContext = await browser.newContext();
    let page = await sellerContext.newPage();

    const seller = unique("seller");
    const buyer = unique("buyer");
    const title = unique("Stethoskop Littmann");

    await registerUser(page, seller);

    await page.getByRole("button", { name: "Open navigation sidebar" }).click();
    await page.getByRole("link", { name: "Flea market" }).first().click();
    await expect(page).toHaveURL(/\/flohmarkt/);
    await clearOverlays(page);

    await page.getByRole("link", { name: "New listing" }).click();
    await expect(page).toHaveURL("/flohmarkt/new");
    await expect(page.getByTestId("phone-required")).toBeVisible();

    await setPhone(page, "0664 1112233");

    await page.goto("/flohmarkt/new");

    await clearOverlays(page);
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

    const sellerPage = page;
    page = await buyerContext.newPage();

    await registerUser(page, buyer);
    await setPhone(page, "+49 151 99887766");

    await page.goto("/flohmarkt");

    await clearOverlays(page);
    await page.getByRole("searchbox").fill(title);
    await page.getByRole("button", { name: "Search" }).click();
    const card = page.getByTestId("listing-card").filter({ hasText: title });
    await expect(card).toBeVisible({ timeout: 10000 });
    await card.getByRole("link", { name: title }).last().click();
    await expect(page).toHaveURL(listingURL);

    await expect(
      page.getByRole("link", { name: "Chat on WhatsApp" }),
    ).toHaveCount(0);

    await page.locator('input[name="amount"]').fill("40");
    await page.locator('input[name="message"]').fill("Abholung am Freitag?");
    await page.getByRole("button", { name: "Send offer" }).click();

    const whatsapp = page
      .getByTestId("listing-contact")
      .getByRole("link", { name: "Chat on WhatsApp" });
    await expect(whatsapp).toBeVisible({ timeout: 10000 });
    await expect(whatsapp).toHaveAttribute(
      "href",
      /^https:\/\/wa\.me\/436641112233\?text=/,
    );
    await expect(page.getByTestId("contact-phone")).toHaveText("+436641112233");

    page = sellerPage;
    await page.goto(listingURL);
    await clearOverlays(page);
    const offers = page.getByTestId("listing-offers");
    await expect(offers).toContainText("Abholung am Freitag?", {
      timeout: 10000,
    });
    await expect(
      offers.getByRole("link", { name: "Chat on WhatsApp" }),
    ).toHaveAttribute("href", /^https:\/\/wa\.me\/4915199887766\?text=/);

    await sellerContext.close();
    await buyerContext.close();
  });

  test("mobile layout", async ({ browser }) => {
    const context = await browser.newContext({
      viewport: { width: 390, height: 844 },
    });
    const page = await context.newPage();

    await registerUser(page, unique("mobile"));
    await page.goto("/flohmarkt");
    await clearOverlays(page);

    const conditionFilter = page.getByRole("combobox", { name: "Condition" });
    await expect(conditionFilter).toBeHidden();
    await page.getByRole("button", { name: "Filters" }).click();
    await expect(conditionFilter).toBeVisible();

    await page
      .getByRole("button", { name: "Instruments & equipment" })
      .click();
    await expect(page).toHaveURL(/category=instruments/);

    const card = page.getByTestId("listing-card").first();
    await expect(card).toBeVisible({ timeout: 10000 });
    await card.getByRole("link").click();
    await expect(page).toHaveURL(/\/flohmarkt\/[a-z0-9]+-/);

    await page.getByRole("link", { name: "Make an offer" }).click();
    await expect(page.locator("#offer")).toBeInViewport();

    await context.close();
  });
});
