import { expect, test } from "@playwright/test";

import { withAdminAccessKey } from "../access_key_admin_assignment";
import {
  attachFiles,
  createThread,
  registerUser,
  replyForm,
  textPDF,
  unique,
} from "../helpers";

const LINES = [
  "Altklausur Zahnerhaltung WS 22/23",
  "Aufgabe 1: Endodontie",
  "Beschreiben Sie die Aufbereitung des Wurzelkanals mit rotierenden Instrumenten.",
  "Nennen Sie typische Komplikationen bei der Spuelung mit Natriumhypochlorit.",
  "Aufgabe 2: Parodontologie",
  "Erklaeren Sie die Einteilung der Parodontitis nach der aktuellen Klassifikation.",
  "Welche Befunde erheben Sie bei der Erstuntersuchung eines neuen Patienten?",
  "Aufgabe 3: Prothetik",
  "Vergleichen Sie Teleskopprothesen und Klammerprothesen hinsichtlich Halt und Hygiene.",
];

test.describe("PDF summary", () => {
  test.beforeAll(async () => {
    await withAdminAccessKey(async ({ adminSettingsUpdate }) => {
      await adminSettingsUpdate({
        services: {
          ai: { pdf_summary: true, pdf_summary_background: false },
        },
      });
    });
  });

  test("summarises a PDF attached to a reply", async ({ page }) => {
    await registerUser(page, unique("pdfsum"));
    await createThread(
      page,
      unique("Altklausur Endo"),
      "Hier ist die Altklausur als PDF.",
    );

    await attachFiles(page, [
      {
        name: "endoklausur.pdf",
        mimeType: "application/pdf",
        buffer: textPDF(LINES),
      },
    ]);

    const form = replyForm(page);
    const editor = form.locator(".ProseMirror[contenteditable='true']");
    await expect(editor.getByText("endoklausur.pdf")).toBeVisible({
      timeout: 15000,
    });
    await form.getByRole("button", { name: "Post" }).click();

    const reply = page
      .locator("li")
      .filter({ hasText: "endoklausur.pdf" })
      .last();
    const button = reply.getByRole("button", { name: "Summarise PDF" });
    await expect(button).toBeVisible({ timeout: 15000 });

    const dialog = page.getByRole("dialog");

    // Text extraction runs right after the upload, so the first attempt may
    // still report that no text is available.
    await expect(async () => {
      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
      await button.click();
      await expect(dialog.getByTestId("pdf-summary")).toBeVisible({
        timeout: 5000,
      });
    }).toPass({ timeout: 30000 });

    await expect(dialog.getByTestId("pdf-summary")).toContainText(
      "An answer for",
    );
    await expect(dialog).toContainText("endoklausur.pdf");

    await page.keyboard.press("Escape");
    await expect(dialog).toHaveCount(0);

    await button.click();
    await expect(dialog.getByTestId("pdf-summary")).toBeVisible();
    await expect(
      dialog.getByText(
        "Quick AI summary from the most relevant lines. A more detailed version is prepared later.",
      ),
    ).toBeVisible();
  });
});
