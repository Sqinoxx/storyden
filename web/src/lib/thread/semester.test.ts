import { test } from "uvu";
import * as assert from "uvu/assert";

import type { ThreadReference } from "@/api/openapi-schema";

import {
  formatTermLabel,
  groupThreadsBySemester,
  parseTermKey,
  parseThreadSemesterMeta,
  selectableTerms,
  termFor,
  termKey,
  threadTerm,
  writeThreadSemesterMeta,
} from "./semester";

function thread(
  partial: Partial<ThreadReference> & { createdAt: string },
): ThreadReference {
  return { pinned: 0, ...partial } as unknown as ThreadReference;
}

test("termFor places months on the right side of both boundaries", () => {
  assert.equal(termFor("2026-03-31T12:00:00Z"), { year: 2025, winter: true });
  assert.equal(termFor("2026-04-01T12:00:00Z"), { year: 2026, winter: false });
  assert.equal(termFor("2026-09-30T12:00:00Z"), { year: 2026, winter: false });
  assert.equal(termFor("2026-10-01T12:00:00Z"), { year: 2026, winter: true });
});

test("termFor accepts a Date as well as an ISO string", () => {
  assert.equal(termFor(new Date(2026, 0, 15)), { year: 2025, winter: true });
});

test("formatTermLabel renders the German shorthand", () => {
  assert.is(formatTermLabel({ year: 2026, winter: false }), "SS26");
  assert.is(formatTermLabel({ year: 2025, winter: true }), "WS25/26");
  assert.is(formatTermLabel({ year: 1999, winter: true }), "WS99/00");
  assert.is(formatTermLabel({ year: 2000, winter: false }), "SS00");
});

test("termKey and parseTermKey round trip", () => {
  for (const term of [
    { year: 2026, winter: false },
    { year: 2025, winter: true },
  ]) {
    assert.equal(parseTermKey(termKey(term)), term);
  }

  assert.is(parseTermKey("2025-ws")?.winter, true);
});

test("parseTermKey rejects malformed input", () => {
  assert.is(parseTermKey("garbage"), undefined);
  assert.is(parseTermKey("2025-XX"), undefined);
  assert.is(parseTermKey("notayear-SS"), undefined);
  assert.is(parseTermKey(undefined), undefined);
  assert.is(parseTermKey(""), undefined);
});

test("selectableTerms runs newest first from one term ahead", () => {
  const terms = selectableTerms(new Date("2026-05-01T00:00:00Z"));

  assert.is(terms.length, 18);
  assert.equal(terms[0], { year: 2026, winter: true });
  assert.equal(terms[1], { year: 2026, winter: false });
  assert.equal(terms[2], { year: 2025, winter: true });
  assert.equal(terms[17], { year: 2018, winter: false });
});

test("parseThreadSemesterMeta tolerates missing and malformed meta", () => {
  assert.equal(parseThreadSemesterMeta({ semester: { term: "2023-WS" } }), {
    year: 2023,
    winter: true,
  });
  assert.is(parseThreadSemesterMeta(undefined), undefined);
  assert.is(parseThreadSemesterMeta({}), undefined);
  assert.is(parseThreadSemesterMeta({ semester: "2023-WS" }), undefined);
  assert.is(parseThreadSemesterMeta({ semester: { term: 42 } }), undefined);
  assert.is(parseThreadSemesterMeta({ semester: { term: "junk" } }), undefined);
});

test("writeThreadSemesterMeta preserves sibling keys", () => {
  const result = writeThreadSemesterMeta(
    { other: "keep", semester: { term: "2023-WS" } },
    { year: 2026, winter: false },
  );

  assert.equal(result, { other: "keep", semester: { term: "2026-SS" } });
});

test("threadTerm falls back to createdAt without meta", () => {
  assert.equal(threadTerm(thread({ createdAt: "2026-05-02T00:00:00Z" })), {
    year: 2026,
    winter: false,
  });

  assert.equal(
    threadTerm(
      thread({
        createdAt: "2026-05-02T00:00:00Z",
        meta: { semester: { term: "2023-WS" } },
      }),
    ),
    { year: 2023, winter: true },
  );
});

test("groupThreadsBySemester orders terms newest first", () => {
  const groups = groupThreadsBySemester([
    thread({ id: "a", createdAt: "2026-05-02T00:00:00Z" }),
    thread({ id: "b", createdAt: "2026-04-02T00:00:00Z" }),
    thread({ id: "c", createdAt: "2025-11-02T00:00:00Z" }),
  ]);

  assert.equal(
    groups.map((g) => g.key),
    ["2026-SS", "2025-WS"],
  );
  assert.is(groups[0]!.threads.length, 2);
  assert.is(groups[1]!.threads.length, 1);
});

test("groupThreadsBySemester collapses a backdated thread into its bucket", () => {
  const groups = groupThreadsBySemester([
    thread({ id: "a", createdAt: "2023-11-02T00:00:00Z" }),
    thread({ id: "b", createdAt: "2026-05-02T00:00:00Z" }),
    thread({
      id: "c",
      createdAt: "2026-05-01T00:00:00Z",
      meta: { semester: { term: "2023-WS" } },
    }),
  ]);

  assert.equal(
    groups.map((g) => g.key),
    ["2026-SS", "2023-WS"],
  );
  assert.equal(
    groups[1]!.threads.map((t) => t.id),
    ["a", "c"],
  );
});

test("groupThreadsBySemester hoists pinned threads into their own group", () => {
  const groups = groupThreadsBySemester([
    thread({ id: "p", createdAt: "2020-11-02T00:00:00Z", pinned: 1 }),
    thread({ id: "a", createdAt: "2026-05-02T00:00:00Z" }),
  ]);

  assert.equal(
    groups.map((g) => g.key),
    ["pinned", "2026-SS"],
  );
  assert.is(groups[0]!.term, null);
});

test.run();
