import { Page, expect, test } from "@playwright/test";

import { dismissOnboarding, registerUser, unique } from "../helpers";

const VIEWPORTS = [
  { name: "iPad mini portrait", width: 768, height: 1024 },
  { name: "iPad Air portrait", width: 820, height: 1180 },
  { name: "iPad Pro 11 portrait", width: 834, height: 1194 },
  { name: "iPad Pro 12.9 portrait", width: 1024, height: 1366 },
  { name: "iPad Air landscape", width: 1180, height: 820 },
  { name: "iPad Pro 12.9 landscape", width: 1366, height: 1024 },
  { name: "Desktop", width: 1440, height: 900 },
];

type Box = { x: number; y: number; width: number; height: number };

async function topbarBoxes(page: Page) {
  return page.evaluate(() => {
    const visible = (el: Element | null) => {
      if (!el) return null;
      const r = el.getBoundingClientRect();
      if (r.width === 0 || r.height === 0) return null;
      return { x: r.x, y: r.y, width: r.width, height: r.height };
    };
    const topbar = document.querySelector(
      "[data-testid='presence-indicator']",
    )?.closest("[class*='topbar']:not([class*='topbar-'])") as Element | null;

    const query = (sel: string) =>
      Array.from(document.querySelectorAll(sel)).find(
        (el) => topbar?.contains(el) && visible(el),
      ) ?? null;

    return {
      topbar: visible(topbar),
      items: {
        title: visible(query("h1 a")),
        theme: visible(query("button[aria-label*='Design']")),
        presence: visible(query("[data-testid='presence-indicator']")),
        compose: visible(query("a[href='/new']")),
        notifications: visible(
          query("[aria-label='Notifications'], [aria-label='Benachrichtigungen']"),
        ),
        account: visible(
          query("[aria-label='Account menu'], [aria-label='Kontomenü']"),
        ),
      },
    };
  });
}

function overlaps(a: Box, b: Box) {
  return (
    a.x < b.x + b.width - 0.5 &&
    b.x < a.x + a.width - 0.5 &&
    a.y < b.y + b.height - 0.5 &&
    b.y < a.y + a.height - 0.5
  );
}

test.describe("Top bar layout", () => {
  for (const language of ["de", "en"] as const) {
    test(`right side controls never overlap (${language})`, async ({
      page,
    }) => {
      await page.setViewportSize({ width: 1440, height: 900 });
      await registerUser(page, unique("topbar"));
      await dismissOnboarding(page);
      await page.evaluate((lang) => {
        localStorage.setItem("storyden-language", lang);
      }, language);
      await page.reload();
      await expect(page.locator("html")).toHaveAttribute("lang", language);

      for (const viewport of VIEWPORTS) {
        await page.setViewportSize(viewport);
        await expect(
          page.getByTestId("presence-indicator").filter({ visible: true }),
        ).toBeVisible();

        const { topbar, items } = await topbarBoxes(page);
        expect(topbar, viewport.name).not.toBeNull();

        const entries = Object.entries(items).filter(
          (entry): entry is [string, Box] => entry[1] !== null,
        );
        expect(
          entries.map(([name]) => name),
          viewport.name,
        ).toEqual(expect.arrayContaining([
            "title",
            "theme",
            "presence",
            "compose",
            "notifications",
            "account",
          ]));

        for (const [name, box] of entries) {
          expect(box.x, `${viewport.name}: ${name} left edge`).toBeGreaterThanOrEqual(
            topbar!.x - 0.5,
          );
          expect(
            box.x + box.width,
            `${viewport.name}: ${name} right edge`,
          ).toBeLessThanOrEqual(topbar!.x + topbar!.width + 0.5);
        }

        for (let i = 0; i < entries.length; i++) {
          for (let j = i + 1; j < entries.length; j++) {
            const [an, a] = entries[i]!;
            const [bn, b] = entries[j]!;
            expect(
              overlaps(a, b),
              `${viewport.name}: ${an} overlaps ${bn}`,
            ).toBe(false);
          }
        }

        expect(
          items.theme!.x + items.theme!.width,
          `${viewport.name}: theme toggle sits left of presence`,
        ).toBeLessThanOrEqual(items.presence!.x);

        const overflow = await page.evaluate(
          () => document.documentElement.scrollWidth > window.innerWidth,
        );
        expect(overflow, `${viewport.name}: horizontal overflow`).toBe(false);

        await page.screenshot({
          path: `test-results/topbar-${language}-${viewport.width}.png`,
          clip: { x: 0, y: 0, width: viewport.width, height: 120 },
        });
      }
    });
  }
});
