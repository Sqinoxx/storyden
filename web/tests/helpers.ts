import { Page, expect } from "@playwright/test";

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
  await page.getByRole("button", { name: "Register" }).click();
  await expectSignedIn(page);
}

export async function login(page: Page, username: string) {
  await page.goto("/login");
  await page.getByRole("textbox", { name: "username" }).fill(username);
  await page.getByRole("textbox", { name: "password" }).fill(PASSWORD);
  await page.getByRole("button", { name: "Login" }).click();
  await expectSignedIn(page);
}

export async function logout(page: Page) {
  await page.getByRole("button", { name: "Account menu" }).click();
  await page.getByRole("menuitem", { name: "Logout" }).click();
  await expect(page.getByRole("link", { name: "Login" })).toBeVisible();
}

export async function createThread(
  page: Page,
  title: string,
  body: string,
): Promise<string> {
  // Several landing pages render more than one "Post" entry point (sidebar and
  // empty-state call to action), and which page sign-in lands on varies.
  await page.getByRole("link", { name: "Post", exact: true }).first().click();
  await expect(page).toHaveURL("/new", { timeout: 5000 });

  await page.locator("#title-input").fill(title);

  // No click first: the floating composer toolbar overlays the editor surface
  // and intercepts pointer events. fill() focuses the element directly, so it
  // does not need a clear hit target.
  const editor = page.locator(".ProseMirror").first();
  await editor.fill(body);

  const post = page.getByRole("button", { name: "Post" });
  await expect(post).toBeEnabled({ timeout: 10000 });
  await post.click();

  await expect(page).toHaveURL(/\/t\//, { timeout: 10000 });
  await dismissOnboarding(page);

  return page.url();
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
