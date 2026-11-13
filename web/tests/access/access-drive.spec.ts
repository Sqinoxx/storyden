import { expect, test } from "@playwright/test";

import { createAdmin, login } from "../access_key_admin_assignment";
import { MOCK_SUB_FOLDER_ID, addFolder } from "../drive/drive_helpers";
import { PASSWORD, dismissOnboarding, registerUser, unique } from "../helpers";

test("drive folders behind a pass stay listed but locked until unlocked", async ({
  page,
  browser,
}) => {
  const seed = unique("accdrive");
  const folderName = `Altprotokolle ${seed}`;
  const passName = `Drive-Paket ${seed}`;

  const adminHandle = unique("kassenwart");
  await createAdmin(page.context(), adminHandle, PASSWORD);
  await login(page, adminHandle, PASSWORD);
  await dismissOnboarding(page);

  await addFolder(page, folderName, MOCK_SUB_FOLDER_ID);

  await page.goto("/access?tab=passes");
  await page.getByRole("button", { name: "Create pass" }).click();
  const editor = page.getByTestId("access-pass-editor");
  await editor.getByRole("textbox", { name: "Name" }).fill(passName);
  await editor.getByText(folderName, { exact: true }).click();
  await editor.getByRole("button", { name: "Save" }).click();
  await expect(page.locator(`[aria-label="${passName}"]`)).toContainText(
    folderName,
    { timeout: 10000 },
  );

  const memberContext = await browser.newContext();
  const member = await memberContext.newPage();
  const memberHandle = unique("student");
  await registerUser(member, memberHandle);

  await member.goto("/drive");
  await dismissOnboarding(member);
  const folderLink = member.getByRole("link", { name: folderName });
  await expect(folderLink.getByTestId("drive-folder-locked-icon")).toBeVisible({
    timeout: 10000,
  });
  await folderLink.click();

  await expect(member.getByRole("region", { name: passName })).toBeVisible({
    timeout: 10000,
  });
  await expect(
    member.getByRole("cell", { name: "handout.pdf", exact: true }),
  ).toHaveCount(0);
  const lockedURL = member.url();

  await page.goto("/access?tab=grants");
  await page.getByRole("button", { name: "Unlock a member" }).click();
  const dialog = page.getByTestId("access-grant-dialog");
  await dialog.getByRole("textbox", { name: "Member" }).fill(memberHandle);
  await dialog
    .getByRole("option", { name: new RegExp(`@${memberHandle}$`) })
    .click();
  await expect(dialog.getByTestId("access-member-selected")).toContainText(
    `@${memberHandle}`,
  );
  await dialog.getByRole("combobox", { name: "Pass" }).selectOption({
    label: passName,
  });
  await dialog.getByRole("button", { name: "Unlock", exact: true }).click();
  await expect(dialog).toHaveCount(0, { timeout: 10000 });

  await member.goto(lockedURL);
  await expect(
    member.getByRole("cell", { name: "handout.pdf", exact: true }),
  ).toBeVisible({ timeout: 10000 });

  await memberContext.close();
});
