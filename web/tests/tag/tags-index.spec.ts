import { Page, expect, test } from "@playwright/test";

import { createThread, registerUser, unique } from "../helpers";

const API = "http://localhost:8001/api";

async function tagThread(page: Page, threadUrl: string, tag: string) {
  const mark = new URL(threadUrl).pathname.split("/").pop();
  const response = await page.request.patch(`${API}/threads/${mark}`, {
    data: { tags: [tag] },
  });
  expect(response.ok()).toBeTruthy();
}

test.describe("Tags index", () => {
  test("members can browse tags and open tagged threads", async ({ page }) => {
    await registerUser(page, unique("tag-reader"));

    const title = unique("Tagged thread");
    const tag = unique("topic").toLowerCase();
    const threadUrl = await createThread(page, title, "Tagged thread body.");
    await tagThread(page, threadUrl, tag);

    const nav = page.getByRole("navigation");
    await expect(
      nav.getByRole("link", { name: "Tag Management" }),
    ).toHaveCount(0);
    await nav.getByRole("link", { name: "Tags", exact: true }).first().click();
    await expect(page).toHaveURL("/tags");

    await expect(
      page.getByRole("button", { name: "Create Tag" }),
    ).toHaveCount(0);

    await page.getByPlaceholder("Search tags...").fill(tag);
    await page.getByRole("link", { name: tag }).click();
    await expect(page).toHaveURL(`/tags/${tag}`);

    await expect(page.getByRole("heading", { name: "Threads" })).toBeVisible();
    await page.getByRole("link", { name: title }).first().click();
    await expect(page).toHaveURL(/\/t\//);
    await expect(page.getByRole("heading", { name: title })).toBeVisible();
  });

  test("members cannot open tag management", async ({ page }) => {
    await registerUser(page, unique("tag-member"));

    await page.goto("/admin/tags");
    await expect(
      page.getByText("Nur für Moderatoren und Admins sichtbar."),
    ).toBeVisible();
    await expect(
      page.getByRole("button", { name: "Create Tag" }),
    ).toHaveCount(0);
  });

  test("guests are asked to log in", async ({ page }) => {
    await page.goto("/tags");
    await expect(page.getByText("Authentication Required")).toBeVisible();
  });
});
