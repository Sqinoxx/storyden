import { test } from "uvu";
import * as assert from "uvu/assert";

import { getLogoURLs } from "./logo";

test("getLogoURLs returns no logo when nothing is set", () => {
  assert.equal(getLogoURLs({ light: "", dark: "" }), { light: "", dark: "" });
  assert.equal(getLogoURLs(undefined), { light: "", dark: "" });
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
