import { Locator, Page, expect } from "@playwright/test";

import { withAdminAccessKey } from "./access_key_admin_assignment";

export const PASSWORD = "TestPassword123!";

/**
 * The suite shares one backend, so every account and thread name needs its own
 * suffix or a retry will collide with the run that created it.
 */
export function unique(prefix: string): string {
  return `${prefix}-${Date.now().toString(36)}-${Math.floor(Math.random() * 1e6).toString(36)}`;
}

export async function dismissOnboarding(page: Page) {
  const skipButton = page.getByRole("button", { name: "Skip" });
  if (await skipButton.isVisible({ timeout: 1000 }).catch(() => false)) {
    await skipButton.click();
  }
}

/**
 * The onboarding checklist and the navigation sidebar can sit over page
 * content and intercept clicks, so close whichever of them is open.
 */
export async function clearOverlays(page: Page) {
  await page.waitForLoadState("networkidle");
  for (const name of ["Skip", "Hide this message", "Close navigation sidebar"]) {
    const button = page.getByRole("button", { name });
    if (await button.first().isVisible({ timeout: 1500 }).catch(() => false)) {
      await button.first().click();
    }
  }
}

/**
 * Where a successful sign-in lands depends on instance configuration (the feed,
 * the discussion index, ...), so the durable signal is "no longer on the auth
 * page, and the session controls are present" rather than a fixed URL.
 */
async function expectSignedIn(page: Page) {
  await expect(page).not.toHaveURL(/\/(login|register)/, { timeout: 10000 });
  await expect(page.getByRole("button", { name: "Account menu" })).toBeVisible({
    timeout: 10000,
  });
  await dismissOnboarding(page);
}

export async function registerUser(page: Page, username: string) {
  await page.goto("/register");
  await page.getByRole("textbox", { name: "username" }).fill(username);
  await page.getByRole("textbox", { name: "password" }).fill(PASSWORD);
  // The handle registration form also requires a Fachsemester (study term)
  // before the native <select>'s required validation lets the form submit.
  await page.getByRole("combobox", { name: "Fachsemester" }).selectOption({ index: 1 });
  await page.getByRole("button", { name: "Register" }).click();
  await expectSignedIn(page);
}

export async function login(page: Page, username: string) {
  await page.goto("/login");
  await page.locator('input[autocomplete="username"]').fill(username);
  await page.locator('input[autocomplete="current-password"]').fill(PASSWORD);
  await page.getByRole("button", { name: /^log ?in$/i }).click();
  await expectSignedIn(page);
}

export async function logout(page: Page) {
  await page.getByRole("button", { name: "Account menu" }).click();
  await page.getByRole("menuitem", { name: "Logout" }).click();
  await page.waitForURL("/login", { timeout: 10000 });
}

let threadCategory: Promise<string> | undefined;

// Members may only publish into a category, so threads go into one shared
// category created once per worker.
export function ensureThreadCategory(): Promise<string> {
  threadCategory ??= withAdminAccessKey(async ({ categoryCreate }) => {
    const slug = unique("threads");
    const name = `Threads ${slug}`;
    await categoryCreate({
      colour: "#3b82f6",
      description: name,
      name,
      slug,
    });
    return name;
  });
  return threadCategory;
}

export async function selectCategory(page: Page, name: string) {
  await page.getByRole("button", { name: "Select category" }).click();
  await page
    .getByTestId("category-tree-panel")
    .locator('[data-part="branch-text"]')
    .filter({ hasText: name })
    .click();
}

// The composer loads categories when it mounts, so call ensureThreadCategory
// before opening it.
export async function selectThreadCategory(page: Page) {
  await selectCategory(page, await ensureThreadCategory());
}

export async function createThread(
  page: Page,
  title: string,
  body: string,
): Promise<string> {
  await ensureThreadCategory();

  // A full load rather than the header link: client navigation reuses the
  // category list cached before the thread category existed.
  await page.goto("/new");
  await dismissOnboarding(page);

  await page.locator("#title-input").fill(title);

  // No click first: the floating composer toolbar overlays the editor surface
  // and intercepts pointer events. fill() focuses the element directly, so it
  // does not need a clear hit target.
  const editor = page.locator(".ProseMirror").first();
  await editor.fill(body);

  await selectThreadCategory(page);

  const post = page.getByRole("button", { name: "Post", exact: true });
  await expect(post).toBeEnabled({ timeout: 10000 });
  await post.click();

  await expect(page).toHaveURL(/\/t\//, { timeout: 10000 });
  await dismissOnboarding(page);

  return page.url();
}

export async function openSidebar(page: Page) {
  const open = page.getByRole("button", { name: "Open navigation sidebar" });
  const close = page.getByRole("button", { name: "Close navigation sidebar" });
  await expect(open.or(close).first()).toBeVisible({ timeout: 10000 });
  if (await open.isVisible()) {
    await open.click();
  }
  await expect(close.first()).toBeVisible();
}

export async function navigateToThread(page: Page, threadUrl: string) {
  await page.goto(threadUrl);
  await dismissOnboarding(page);
}

export function replyForm(page: Page) {
  return page
    .locator("form")
    .filter({ has: page.getByRole("button", { name: "Post" }) })
    .last();
}

/**
 * Attaches files through the composer's attach control.
 *
 * The rich editor menu renders its own file input inside the same form, so
 * addressing the input directly is ambiguous. Going through the chooser also
 * exercises what a user actually does, and proves the control is not disabled.
 */
export async function attachFiles(
  page: Page,
  files: { name: string; mimeType: string; buffer: Buffer }[],
) {
  const attach = replyForm(page)
    .locator("label")
    .filter({ hasText: /Upload file|Datei hochladen/ })
    .first();

  await expect(attach).toBeVisible({ timeout: 10000 });

  const [chooser] = await Promise.all([
    page.waitForEvent("filechooser"),
    attach.click(),
  ]);

  await chooser.setFiles(files);
}

/**
 * The quick share composer on the feed. It is the only form on that page that
 * embeds a rich text editor, and its post button label is localised, so the
 * editor is the stable anchor.
 */
export function quickShareForm(page: Page) {
  return page
    .locator("form")
    .filter({ has: page.locator("[id^='rich-text-editor-']") })
    .first();
}

/**
 * Drags files in from outside the page and drops them on an element.
 *
 * Playwright cannot originate an OS-level file drag, so the DataTransfer is
 * built inside the page and carried through the whole enter/over/drop sequence
 * the way a real drag does — the composer tracks enter/leave to decide whether
 * a drop is a file drop at all, so dispatching drop alone would not exercise it.
 */
export async function dropFiles(
  page: Page,
  target: Locator,
  files: { name: string; mimeType: string; buffer: Buffer }[],
) {
  const payload = files.map((f) => ({
    name: f.name,
    mimeType: f.mimeType,
    base64: f.buffer.toString("base64"),
  }));

  const dataTransfer = await page.evaluateHandle((items) => {
    const dt = new DataTransfer();

    for (const item of items) {
      const binary = atob(item.base64);
      const bytes = new Uint8Array(binary.length);
      for (let i = 0; i < binary.length; i++) {
        bytes[i] = binary.charCodeAt(i);
      }
      dt.items.add(new File([bytes], item.name, { type: item.mimeType }));
    }

    return dt;
  }, payload);

  await target.dispatchEvent("dragenter", { dataTransfer });
  await target.dispatchEvent("dragover", { dataTransfer });
  await target.dispatchEvent("drop", { dataTransfer });
}

export async function dropURL(page: Page, target: Locator, url: string) {
  const dataTransfer = await page.evaluateHandle((href) => {
    const dt = new DataTransfer();
    dt.setData("text/uri-list", href);
    dt.setData("text/plain", href);
    return dt;
  }, url);

  const box = await target.boundingBox();
  const coords = box
    ? { clientX: box.x + box.width / 2, clientY: box.y + box.height / 2 }
    : {};

  await target.dispatchEvent("dragenter", { dataTransfer, ...coords });
  await target.dispatchEvent("dragover", { dataTransfer, ...coords });
  await target.dispatchEvent("drop", { dataTransfer, ...coords });
}

export async function postReply(page: Page, body: string) {
  const form = page.locator("form").filter({ hasText: "Reply to" });
  const editor = form.locator(".ProseMirror[contenteditable='true']");
  await editor.fill(body);

  await form.getByRole("button", { name: "Post" }).click();

  await page.locator("li").filter({ hasText: body }).waitFor({ timeout: 10000 });
}

/**
 * A real, decodable 1x1 PNG. The backend sniffs content type from the bytes on
 * upload, so a placeholder string would be stored as text/plain and then get
 * served as an attachment rather than an image.
 */
export const ONE_PIXEL_PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mP8z8BQDwAEhQGAhKmMIQAAAABJRU5ErkJggg==",
  "base64",
);

/**
 * A single page PDF with a real text layer, so the backend extracts its text
 * without OCR binaries. Offsets in the xref table must match the bytes.
 */
export function textPDF(lines: string[]): Buffer {
  const escape = (s: string) => s.replace(/[\\()]/g, (c) => `\\${c}`);
  const content =
    "BT /F1 11 Tf 50 780 Td 14 TL " +
    lines.map((l) => `(${escape(l)}) '`).join(" ") +
    " ET";
  const objects = [
    "<< /Type /Catalog /Pages 2 0 R >>",
    "<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
    "<< /Type /Page /Parent 2 0 R /MediaBox [0 0 595 842] /Contents 4 0 R /Resources << /Font << /F1 5 0 R >> >> >>",
    `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
    "<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
  ];

  let out = "%PDF-1.4\n";
  const offsets: number[] = [];
  objects.forEach((body, i) => {
    offsets.push(out.length);
    out += `${i + 1} 0 obj\n${body}\nendobj\n`;
  });
  const xref = out.length;
  out += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n`;
  for (const o of offsets) {
    out += `${String(o).padStart(10, "0")} 00000 n \n`;
  }
  out += `trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF\n`;
  return Buffer.from(out, "latin1");
}
