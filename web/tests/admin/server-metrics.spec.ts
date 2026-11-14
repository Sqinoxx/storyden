import { expect, test } from "@playwright/test";

import { createAdmin, login } from "../access_key_admin_assignment";

const PASSWORD = "TestPassword123!";

test("admin sees live server load in admin tab and account menu", async ({
  page,
}, testInfo) => {
  const adminHandle = `server-admin-${Date.now().toString(36)}`;

  await createAdmin(page.context(), adminHandle, PASSWORD);
  await login(page, adminHandle, PASSWORD);

  await page.goto("/admin");
  await expect(
    page.getByRole("tab", { name: "Server", exact: true }),
  ).toBeVisible();
  const hideOnboarding = page.getByRole("button", {
    name: "Hide this message",
  });
  if (await hideOnboarding.isVisible()) {
    await hideOnboarding.click();
  }
  await page.getByRole("tab", { name: "Server", exact: true }).click();
  await expect(page).toHaveURL(/tab=server/);

  await expect(
    page.getByRole("heading", { name: "Server-Auslastung" }),
  ).toBeVisible();

  const kpis = page.getByTestId("server-kpis");
  for (const label of [
    "Prozessor",
    "Arbeitsspeicher",
    "Festplatte",
    "Anfragen pro Sekunde",
    "Antwortzeit",
    "Datenbank",
  ]) {
    await expect(kpis.getByText(label, { exact: true })).toBeVisible();
  }
  await expect(kpis.getByText(/\d+ %/).first()).toBeVisible();

  await expect(kpis.getByText(/\d+ Kerne/)).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Prozessor & Arbeitsspeicher" }),
  ).toBeVisible();

  await expect(page.getByText(/Ein Messpunkt alle 5 Sekunden/)).toBeVisible();
  await expect(page.locator(".recharts-line").first()).toBeVisible({
    timeout: 20000,
  });
  await page.screenshot({
    path: testInfo.outputPath("server-desktop.png"),
    fullPage: true,
  });

  await page.getByRole("button", { name: "24 Std" }).click();
  await expect(page.getByText(/Minutenmittelwerte/)).toBeVisible();

  await page
    .getByRole("button", { name: /account menu|kontomenü/i })
    .filter({ visible: true })
    .first()
    .click();
  const badge = page
    .getByTestId("server-load-badge")
    .filter({ visible: true })
    .first();
  await expect(badge.getByText("CPU")).toBeVisible();
  await expect(badge.getByText(/^\d+ %$/).first()).toBeVisible();
});
