import { expect, test } from "@playwright/test";

import {
  ONE_PIXEL_PNG,
  attachFiles,
  createThread,
  registerUser,
  replyForm,
  unique,
} from "../helpers";

const PDF = Buffer.from(
  "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[]/Count 0>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n",
  "utf8",
);

test.describe("Attachment upload", () => {
  test("uploads an image and a document, then serves them back", async ({
    page,
  }) => {
    await registerUser(page, unique("uploader"));
    await createThread(
      page,
      unique("Thread with attachments"),
      "This thread receives attachments.",
    );

    await attachFiles(page, [
      { name: "screenshot.png", mimeType: "image/png", buffer: ONE_PIXEL_PNG },
      { name: "handout.pdf", mimeType: "application/pdf", buffer: PDF },
    ]);

    const form = replyForm(page);
    const editor = form.locator(".ProseMirror[contenteditable='true']");

    // The composer body is populated from the upload responses, so waiting on
    // the rendered markup is also waiting for both uploads to finish.
    await expect(editor.locator("img")).toBeVisible({ timeout: 15000 });
    await expect(
      editor.locator("a").filter({ hasText: "handout.pdf" }),
    ).toBeVisible({ timeout: 15000 });

    const imageSrc = await editor.locator("img").getAttribute("src");
    expect(imageSrc).toBeTruthy();

    await form.getByRole("button", { name: "Post" }).click();

    const reply = page.locator("li").filter({ has: page.locator("img") }).last();
    await expect(reply).toBeVisible({ timeout: 15000 });

    // The asset must actually be downloadable, with the headers the browser
    // and any cache in front of it rely on.
    const asset = await page.request.get(imageSrc!);
    expect(asset.status()).toBe(200);
    expect(asset.headers()["content-type"]).toContain("image/png");
    expect(asset.headers()["x-content-type-options"]).toBe("nosniff");
    expect(asset.headers()["etag"]).toBeTruthy();
    expect(asset.headers()["accept-ranges"]).toBe("bytes");
  });

  test("reports a failed upload instead of silently posting without it", async ({
    page,
  }) => {
    await registerUser(page, unique("upload-failer"));
    await createThread(
      page,
      unique("Thread with a broken upload"),
      "This thread tests upload failure reporting.",
    );

    await page.route("**/api/assets?*", (route) =>
      route.fulfill({
        status: 500,
        contentType: "application/json",
        body: JSON.stringify({ error: "storage unavailable" }),
      }),
    );

    await attachFiles(page, [
      { name: "doomed.png", mimeType: "image/png", buffer: ONE_PIXEL_PNG },
    ]);

    // A bare catch used to swallow this, leaving the user to post a reply that
    // silently lost its attachment.
    await expect(
      page.getByText(/storage unavailable|failed|error/i).first(),
    ).toBeVisible({ timeout: 15000 });

    // Nothing was attached, and the control is usable again for a retry.
    await expect(replyForm(page).locator(".ProseMirror img")).toHaveCount(0);
    await attachFiles(page, [
      { name: "retry.png", mimeType: "image/png", buffer: ONE_PIXEL_PNG },
    ]);
  });
});
