import { test } from "uvu";
import * as assert from "uvu/assert";

import { FALLBACK_LOGO_URL, getLogoURLs } from "./logo";

test("getLogoURLs falls back to the bundled logo when nothing is set", () => {
  assert.equal(getLogoURLs({ light: "", dark: "" }), {
    light: FALLBACK_LOGO_URL,
    dark: FALLBACK_LOGO_URL,
  });
  assert.equal(getLogoURLs(undefined), {
    light: FALLBACK_LOGO_URL,
    dark: FALLBACK_LOGO_URL,
  });
});

test("getLogoURLs uses a single uploaded logo for both modes", () => {
  const urls = getLogoURLs({ light: "", dark: "/assets/dark.png" });
  assert.ok(urls.light.endsWith("/assets/dark.png"));
  assert.equal(urls.light, urls.dark);
});

test("getLogoURLs keeps separate light and dark logos", () => {
  const urls = getLogoURLs({
    light: "/assets/light.png",
    dark: "/assets/dark.png",
  });
  assert.ok(urls.light.endsWith("/assets/light.png"));
  assert.ok(urls.dark.endsWith("/assets/dark.png"));
});

test.run();
