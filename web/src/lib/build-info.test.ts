import { test } from "uvu";
import * as assert from "uvu/assert";

import { formatBuildInfo } from "./build-info";

test("formats full build info", () => {
  assert.is(
    formatBuildInfo({ number: "3396", commit: "006cb8b", date: "2026-11-11" }),
    "Build 3396 · 006cb8b · 2026-11-11",
  );
});

test("omits missing parts", () => {
  assert.is(formatBuildInfo({ number: "3396" }), "Build 3396");
});

test("returns undefined without build info", () => {
  assert.is(formatBuildInfo(undefined), undefined);
});

test.run();
