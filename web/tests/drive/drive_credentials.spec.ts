import { generateKeyPairSync } from "node:crypto";
import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";

import { expect, test } from "@playwright/test";

import { createAdmin, login } from "../access_key_admin_assignment";
import { dismissOnboarding, unique } from "../helpers";

const PASSWORD = "TestPassword123!";

// A syntactically valid Google service account key with a throwaway RSA key,
// so the real credential-parsing path in the backend is exercised without
// talking to Google.
function writeFakeServiceAccountKey(email: string): string {
  const { privateKey } = generateKeyPairSync("rsa", {
    modulusLength: 2048,
    privateKeyEncoding: { type: "pkcs8", format: "pem" },
    publicKeyEncoding: { type: "spki", format: "pem" },
  });

  const key = {
    type: "service_account",
    project_id: "storyden-test",
    private_key_id: "test-key-id",
    private_key: privateKey,
    client_email: email,
    client_id: "000000000000000000000",
    token_uri: "https://oauth2.googleapis.com/token",
  };

  const dir = mkdtempSync(join(tmpdir(), "storyden-drive-key-"));
  const path = join(dir, "service-account.json");
  writeFileSync(path, JSON.stringify(key));

  return path;
}

test.describe("google drive credentials", () => {
  test("admin uploads and removes a service account key", async ({
    page,
  }) => {
    const suffix = unique("drivecreds");
    const admin = `admin${suffix}`;
    const email = `storyden-${suffix}@example-project.iam.gserviceaccount.com`;

    await createAdmin(page.context(), admin, PASSWORD);
    await login(page, admin, PASSWORD);
    await dismissOnboarding(page);

    await page.goto("/admin?tab=drive");
    await expect(
      page.getByRole("heading", { name: "Google Drive access" }),
    ).toBeVisible();

    const keyPath = writeFakeServiceAccountKey(email);
    await page.locator('input[type="file"]').setInputFiles(keyPath);

    await expect(
      page.getByText("Using an uploaded service account key."),
    ).toBeVisible();
    await expect(page.getByText(email)).toBeVisible();

    await page.getByRole("button", { name: "Remove key" }).click();
    await page
      .getByRole("button", { name: /are you sure/i })
      .click();

    await expect(
      page.getByText("Using an uploaded service account key."),
    ).toBeHidden();
  });
});
