import { test } from "uvu";
import * as assert from "uvu/assert";
import { API_ADDRESS } from "@/config";

import {
  getAssetURL,
  getCleanFilename,
  normalizeAssetPath,
  normalizeFilename,
  rebaseAssetURL,
} from "./asset";

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

test("getCleanFilename formats slugified hyphenated filenames nicely", () => {
  assert.is(
    getCleanFilename("d9ihuvuqbbt08i08p1j0_belegungen-drucken-1.pdf"),
    "belegungen drucken 1.pdf"
  );
  assert.is(
    getCleanFilename("d9ihuvuqbbt08i08p1j0-belegungen-drucken-1-pdf"),
    "belegungen drucken 1.pdf"
  );
  assert.is(
    getCleanFilename("belegungen drucken (1).pdf"),
    "belegungen drucken (1).pdf"
  );
});

test("normalizeFilename normalizes cleaned filename", () => {
  assert.is(normalizeFilename("d9ihuvuqbbt08i08p1j0_Untitled.pdf"), "untitledpdf");
});

test("rebaseAssetURL moves asset links from a previous domain onto the API address", () => {
  assert.is(
    rebaseAssetURL("https://unidentists.social/api/assets/d9ihuvuqbbt08i08p1j0-z1-pdf"),
    `${API_ADDRESS}/api/assets/d9ihuvuqbbt08i08p1j0-z1-pdf`
  );
  assert.is(
    rebaseAssetURL("http://192.168.178.30:8000/api/assets/a.pdf?filename=A.pdf#page=2"),
    `${API_ADDRESS}/api/assets/a.pdf?filename=A.pdf#page=2`
  );
});

test("rebaseAssetURL leaves relative and non-asset URLs alone", () => {
  assert.is(rebaseAssetURL("/api/assets/a.pdf"), "/api/assets/a.pdf");
  assert.is(
    rebaseAssetURL("https://example.com/files/a.pdf"),
    "https://example.com/files/a.pdf"
  );
  assert.is(
    rebaseAssetURL("https://example.com/other/api/assets/a.pdf"),
    "https://example.com/other/api/assets/a.pdf"
  );
});

test("getAssetURL rebases absolute asset paths", () => {
  assert.is(
    getAssetURL("https://unidentists.social/api/assets/a.pdf"),
    `${API_ADDRESS}/api/assets/a.pdf`
  );
  assert.is(getAssetURL("/api/assets/a.pdf"), `${API_ADDRESS}/api/assets/a.pdf`);
});

test.run();
