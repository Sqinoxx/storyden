import { test } from "uvu";
import * as assert from "uvu/assert";

import type { AccessGrant, AccessPass } from "@/api/openapi-schema";

import {
  categoryAccess,
  categoryAncestry,
  lockedCategoryIDs,
  matchesGrantQuery,
  membershipAction,
  passStats,
} from "./access";

const categories = [
  { id: "z1" },
  { id: "anatomy", parent: "z1" },
  { id: "histo", parent: "anatomy" },
  { id: "open" },
];

function pass(
  id: string,
  categoryIDs: string[],
  extra: Partial<AccessPass> = {},
): AccessPass {
  return {
    id,
    name: id,
    price_label: "10 €",
    colour: "#000",
    description: "",
    payment_info: "",
    sort: 0,
    requestable: true,
    category_ids: categoryIDs,
    drive_folder_ids: [],
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...extra,
  };
}

test("ancestry walks to the root", () => {
  assert.equal(categoryAncestry(categories, "histo"), [
    "histo",
    "anatomy",
    "z1",
  ]);
});

test("ancestry survives cycles", () => {
  const cyclic = [
    { id: "a", parent: "b" },
    { id: "b", parent: "a" },
  ];
  assert.equal(categoryAncestry(cyclic, "a"), ["a", "b"]);
});

test("locks are inherited by subcategories", () => {
  const passes = [pass("z1", ["z1"])];
  assert.is(categoryAccess(passes, categories, "histo", false).state, "locked");
  assert.is(categoryAccess(passes, categories, "open", false).state, "open");
});

test("holding any covering pass unlocks", () => {
  const passes = [
    pass("z1", ["z1"]),
    pass("bundle", ["z1"], {
      membership: { status: "active", reference_code: "B-AAAA" },
    }),
  ];
  const access = categoryAccess(passes, categories, "anatomy", false);
  assert.is(access.state, "unlocked");
  assert.is(access.state === "unlocked" && access.held.id, "bundle");
});

test("expired or requested memberships stay locked", () => {
  for (const status of [
    "expired",
    "requested",
    "rejected",
    "revoked",
  ] as const) {
    const passes = [
      pass("z1", ["z1"], { membership: { status, reference_code: "Z1-AAAA" } }),
    ];
    assert.is(
      categoryAccess(passes, categories, "z1", false).state,
      "locked",
      status,
    );
  }
});

test("staff bypass hides locks", () => {
  const passes = [pass("z1", ["z1"])];
  assert.is(categoryAccess(passes, categories, "z1", true).state, "open");
  assert.equal(lockedCategoryIDs(passes, categories, true).size, 0);
  assert.equal([...lockedCategoryIDs(passes, categories, false)].sort(), [
    "anatomy",
    "histo",
    "z1",
  ]);
});

test("membership actions", () => {
  assert.is(membershipAction(pass("p", []), false), "login");
  assert.is(membershipAction(pass("p", []), true), "request");
  assert.is(
    membershipAction(pass("p", [], { requestable: false }), true),
    "none",
  );
  assert.is(
    membershipAction(
      pass("p", [], {
        membership: { status: "requested", reference_code: "x" },
      }),
      true,
    ),
    "withdraw",
  );
  assert.is(
    membershipAction(
      pass("p", [], { membership: { status: "active", reference_code: "x" } }),
      true,
    ),
    "none",
  );
  assert.is(
    membershipAction(
      pass("p", [], {
        membership: { status: "rejected", reference_code: "x" },
      }),
      true,
    ),
    "request",
  );
});

function grant(
  passID: string,
  status: AccessGrant["status"],
  extra: Partial<AccessGrant> = {},
): AccessGrant {
  return {
    id: `${passID}-${status}`,
    pass: { id: passID, name: passID, price_label: "", colour: "#000" },
    account: {
      id: "acc",
      handle: "baldur",
      name: "Baldur Odinson",
      joined: "2026-01-01T00:00:00Z",
      roles: [],
    },
    status,
    reference_code: "SD-7K3Q",
    note: "",
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    ...extra,
  };
}

test("grant search matches name, handle and reference", () => {
  const g = grant("z1", "requested");
  assert.ok(matchesGrantQuery(g, ""));
  assert.ok(matchesGrantQuery(g, "odin"));
  assert.ok(matchesGrantQuery(g, "@BALD"));
  assert.ok(matchesGrantQuery(g, "7k3q"));
  assert.not.ok(matchesGrantQuery(g, "loki"));
});

test("pass stats count per pass and flag soon expiring grants", () => {
  const now = new Date("2026-11-15T12:00:00Z");
  const stats = passStats(
    [
      grant("z1", "active", { expires_at: "2026-11-20T00:00:00Z" }),
      grant("z1", "active", { expires_at: "2027-03-31T00:00:00Z" }),
      grant("z1", "active"),
      grant("z1", "requested"),
      grant("z1", "expired"),
      grant("z1", "revoked"),
      grant("z2", "requested"),
    ],
    now,
  );

  assert.equal(stats.get("z1"), {
    active: 3,
    requested: 1,
    expiring: 1,
    expired: 1,
  });
  assert.equal(stats.get("z2"), {
    active: 0,
    requested: 1,
    expiring: 0,
    expired: 0,
  });
});

test.run();
