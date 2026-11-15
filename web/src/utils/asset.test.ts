import { test } from "uvu";
import * as assert from "uvu/assert";
import { API_ADDRESS } from "@/config";

import {
  extractBodyAttachmentRefs,
  getAssetURL,
  getCleanFilename,
  isAssetEmbeddedInBody,
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


test("extractBodyAttachmentRefs reads file attachment links without a DOM", () => {
  const body =
    '<p>Hi <a href="https://old.example/api/assets/d9n2uhh3fmss73bsumm0-ph2-altfragen-totale-pdf" data-filename="PH2 Altfragen Totale.pdf" data-type="file-attachment" target="_blank" download="PH2 Altfragen Totale.pdf">PH2 Altfragen Totale.pdf</a>' +
    '<a href="/api/assets/abc" data-filename="BR&Uuml;CKEN &amp; Co.pdf">x</a>' +
    '<span data-type="file-attachment" src="/api/assets/span-file">\n<b>Span&#39;s file.pdf</b></span>' +
    '<span class="mention">ignored</span></p>';

  assert.equal(extractBodyAttachmentRefs(body), [
    {
      href: "https://old.example/api/assets/d9n2uhh3fmss73bsumm0-ph2-altfragen-totale-pdf",
      filename: "PH2 Altfragen Totale.pdf",
    },
    { href: "/api/assets/abc", filename: "BR&Uuml;CKEN & Co.pdf" },
    { href: "/api/assets/span-file", filename: "Span's file.pdf" },
  ]);
  assert.equal(extractBodyAttachmentRefs(undefined), []);
});

test("isAssetEmbeddedInBody matches by path, id or filename", () => {
  const refs = extractBodyAttachmentRefs(
    '<a href="https://x/api/assets/d9n2uhh3fmss73bsumm0-ph2-altfragen-totale-pdf" data-filename="PH2 Altfragen Totale.pdf">PH2 Altfragen Totale.pdf</a>' +
      '<a href="/api/assets/other" data-filename="Phantom II &#8211; Fragen.pdf">y</a>',
  );

  assert.ok(
    isAssetEmbeddedInBody(
      {
        id: "d9n2uhh3fmss73bsumm0",
        path: "assets/d9n2uhh3fmss73bsumm0-ph2-altfragen-totale-pdf",
        filename: "d9n2uhh3fmss73bsumm0-ph2-altfragen-totale-pdf",
      },
      refs,
    ),
  );
  assert.ok(
    isAssetEmbeddedInBody(
      { id: "zzz", path: "zzz", filename: "Phantom II – Fragen.pdf" },
      refs,
    ),
  );
  assert.not.ok(
    isAssetEmbeddedInBody(
      { id: "nope", path: "nope", filename: "Sonstige.pdf" },
      refs,
    ),
  );
});

test.run();
