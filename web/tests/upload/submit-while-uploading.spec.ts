import { Page, Route, expect, test } from "@playwright/test";

import {
  createThread,
  dismissOnboarding,
  dropFiles,
  registerUser,
  unique,
} from "../helpers";

const PDF = Buffer.from(
  "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[]/Count 0>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n",
  "utf8",
);

async function holdUploads(page: Page) {
  let release!: () => void;
  const released = new Promise<void>((resolve) => (release = resolve));

  await page.route("**/api/assets?*", async (route: Route) => {
    await released;
    await route.continue();
  });

  return release;
}

test.describe("Submitting while an upload is in flight", () => {
  test("thread edit holds save until the dropped file has uploaded", async ({
    page,
  }) => {
    await registerUser(page, unique("editupload"));
    const url = await createThread(
      page,
      unique("Thread edited with upload"),
      "This thread gets a file while editing.",
    );

    await page.goto(`${url}?edit=true`);
    await dismissOnboarding(page);

    const form = page
      .locator("form")
      .filter({ has: page.getByRole("button", { name: "Save" }) })
      .first();
    const save = form.getByRole("button", { name: "Save" });
    const composer = form.locator("[id^='rich-text-editor-']").first();
    await expect(composer).toBeVisible({ timeout: 10000 });
    await expect(save).toBeEnabled();

    const release = await holdUploads(page);

    await dropFiles(page, composer, [
      { name: "altklausur.pdf", mimeType: "application/pdf", buffer: PDF },
    ]);

    await expect(save).toBeDisabled({ timeout: 5000 });

    release();

    await expect(form.getByText(/altklausur/i).first()).toBeVisible({
      timeout: 15000,
    });
    await expect(save).toBeEnabled({ timeout: 15000 });
    await save.click();

    await expect(page).not.toHaveURL(/edit=true/, { timeout: 10000 });

    await page.reload();
    await dismissOnboarding(page);
    await expect(page.getByText(/altklausur/i).first()).toBeVisible({
      timeout: 15000,
    });
  });

  test("new post holds publish until the dropped file has uploaded", async ({
    page,
  }) => {
    await registerUser(page, unique("composeupload"));

    await page.getByRole("link", { name: "Post", exact: true }).first().click();
    await expect(page).toHaveURL("/new", { timeout: 5000 });

    await page.locator("#title-input").fill(unique("Post with upload"));
    await page.locator(".ProseMirror").first().fill("Body with a file.");

    const post = page.getByRole("button", { name: "Post" });
    await expect(post).toBeEnabled({ timeout: 10000 });

    const release = await holdUploads(page);

    await dropFiles(page, page.locator("[id^='rich-text-editor-']").first(), [
      { name: "skript.pdf", mimeType: "application/pdf", buffer: PDF },
    ]);

    await expect(post).toBeDisabled({ timeout: 5000 });

    release();

    await expect(post).toBeEnabled({ timeout: 15000 });
    await post.click();

    await expect(page).toHaveURL(/\/t\//, { timeout: 10000 });
    await dismissOnboarding(page);
    await expect(page.getByText(/skript/i).first()).toBeVisible({
      timeout: 15000,
    });
  });
});
