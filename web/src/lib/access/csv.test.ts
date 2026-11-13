import { test } from "uvu";
import * as assert from "uvu/assert";

import type { AccessGrant } from "@/api/openapi-schema";

import { csvCell, grantsToCsv } from "./csv";

const labels = {
  name: "Name",
  handle: "Handle",
  pass: "Pass",
  reference: "Code",
  status: "Status",
  requested: "Requested",
  decided: "Decided",
  expires: "Expires",
  decidedBy: "By",
  note: "Note",
  reason: "Reason",
};

function grant(overrides: Partial<AccessGrant> = {}): AccessGrant {
  return {
    id: "g1",
    account: { id: "a1", handle: "baldur", name: "Baldur" },
    pass: { id: "p1", name: "Z1", price_label: "10 €", colour: "" },
    status: "active",
    reference_code: "Z1-ABCD",
    note: "",
    createdAt: "2026-10-01T10:00:00Z",
    updatedAt: "2026-10-01T10:00:00Z",
    ...overrides,
  } as AccessGrant;
}

test("plain values pass through", () => {
  assert.is(csvCell("Baldur"), "Baldur");
});

test("separators, quotes and line breaks are quoted", () => {
  assert.is(csvCell("a;b"), '"a;b"');
  assert.is(csvCell('sagt "hi"'), '"sagt ""hi"""');
  assert.is(csvCell("zwei\nzeilen"), '"zwei\nzeilen"');
});

test("formula-looking values are neutralised", () => {
  assert.is(csvCell("=HYPERLINK(1)"), "'=HYPERLINK(1)");
  assert.is(csvCell("@sum"), "'@sum");
  assert.is(csvCell("-5"), "'-5");
});

test("rows carry the grant with dates as YYYY-MM-DD", () => {
  const csv = grantsToCsv(
    [
      grant({
        requested_at: "2026-10-01T08:00:00Z",
        decided_at: "2026-10-03T09:00:00Z",
        expires_at: "2027-03-31T21:59:59Z",
        decided_by: {
          id: "a2",
          handle: "odin",
          name: "Odin",
        } as AccessGrant["account"],
        note: "10 € bar",
      }),
      grant({ status: "rejected", reason: "keine Zahlung; sorry" }),
    ],
    labels,
    (s) => s.toUpperCase(),
  );

  assert.ok(csv.startsWith("﻿"));
  const lines = csv.slice(1).trimEnd().split("\r\n");
  assert.is(lines.length, 3);
  assert.is(
    lines[0],
    "Name;Handle;Pass;Code;Status;Requested;Decided;Expires;By;Note;Reason",
  );
  assert.is(
    lines[1],
    "Baldur;baldur;Z1;Z1-ABCD;ACTIVE;2026-10-01;2026-10-03;2027-03-31;Odin;10 € bar;",
  );
  assert.is(
    lines[2],
    'Baldur;baldur;Z1;Z1-ABCD;REJECTED;;;;;;"keine Zahlung; sorry"',
  );
});

test.run();
