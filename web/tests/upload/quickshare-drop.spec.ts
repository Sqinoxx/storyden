import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import {
  ONE_PIXEL_PNG,
  dismissOnboarding,
  dropFiles,
  quickShareForm,
  registerUser,
  unique,
} from "../helpers";

const PDF = Buffer.from(
  "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Kids[]/Count 0>>endobj\ntrailer<</Root 1 0 R>>\n%%EOF\n",
  "utf8",
);

test.describe("Quick share drag and drop", () => {
  test("uploads dropped files into the composer body", async ({ page }) => {
    // Quick share only renders where a plain member is allowed to post, which
    // is a leaf category — the uncategorised feed needs PostInAnyCategory.
    const seed = unique("qsdrop");
    const slug = `qs-drop-${seed}`;

    await withAdminAccessKey(async ({ categoryCreate }) => {
      await categoryCreate({
        colour: "#3b82f6",
        description: `Quick share drop target ${seed}`,
        name: `Quick share drop ${seed}`,
        slug,
      });
    });

    await registerUser(page, unique("dropper").replace(/-/g, ""));
    await page.goto(`/d/${slug}`);
    await dismissOnboarding(page);

    const form = quickShareForm(page);
    const composer = form.locator("[id^='rich-text-editor-']").first();
    await expect(composer).toBeVisible({ timeout: 10000 });

    await dropFiles(page, composer, [
      { name: "handout.pdf", mimeType: "application/pdf", buffer: PDF },
      { name: "screenshot.png", mimeType: "image/png", buffer: ONE_PIXEL_PNG },
    ]);

    const editor = form.locator(".ProseMirror");

    // Documents never enter the document on their own, so they only reach the
    // post through the attachment node the composer inserts. Without that the
    // upload still succeeds and the file silently never appears anywhere.
    await expect(editor.getByText("handout.pdf")).toBeVisible({
      timeout: 15000,
    });

    // Images go in as a blob placeholder first, so the durable signal that the
    // upload survived to completion is the src being swapped for the asset URL.
    const image = editor.locator('img[alt="screenshot.png"]');
    await expect(image).toBeVisible({ timeout: 15000 });
    await expect(image).toHaveAttribute("src", /^https?:\/\//, {
      timeout: 15000,
    });
  });
});
