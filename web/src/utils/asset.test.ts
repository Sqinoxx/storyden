import { test } from "uvu";
import * as assert from "uvu/assert";
import { getCleanFilename, normalizeFilename, normalizeAssetPath } from "./asset";

test("getCleanFilename removes 20-character xid prefix with underscore", () => {
  assert.is(getCleanFilename("d9ihuvuqbbt08i08p1j0_untitled"), "untitled");
  assert.is(getCleanFilename("d9ihuvuqbbt08i08p1j0_untitled.pdf"), "untitled.pdf");
});

test("getCleanFilename removes 20-character xid prefix with hyphen", () => {
  assert.is(getCleanFilename("d9ihuvuqbbt08i08p1j0-document.docx"), "document.docx");
  assert.is(getCleanFilename("d9n2uhh3fmss73bsumm0-foo-pdf"), "foo.pdf");
});

test("getCleanFilename handles full asset paths and query parameters", () => {
  assert.is(
    getCleanFilename("/api/assets/d9ihuvuqbbt08i08p1j0_rechnung_2026.pdf?v=1"),
    "rechnung_2026.pdf"
  );
});

test("normalizeFilename normalizes cleaned filename", () => {
  assert.is(normalizeFilename("d9ihuvuqbbt08i08p1j0_Untitled.pdf"), "untitledpdf");
});

test.run();
