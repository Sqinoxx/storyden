import { test } from "uvu";
import * as assert from "uvu/assert";

import type { Category } from "@/api/openapi-schema";

import {
  buildCategoryTree,
  findCategoryNode,
  findCategoryPath,
  isDescendant,
  isLeafCategory,
} from "./tree";

function category(
  id: string,
  sort: number,
  parent?: string,
  name: string = id,
): Category {
  return {
    id,
    createdAt: "2026-01-01T00:00:00Z",
    updatedAt: "2026-01-01T00:00:00Z",
    children: [],
    colour: "#000000",
    description: "",
    name,
    slug: id,
    sort,
    postCount: 0,
    parent,
  } as Category;
}

test("buildCategoryTree sorts roots and children by sort", () => {
  const categories = [
    category("child-b", 20, "root-a"),
    category("root-b", 30),
    category("child-a", 10, "root-a"),
    category("root-a", 10),
  ];

  const tree = buildCategoryTree(categories);

  assert.equal(
    tree.map((c) => c.id),
    ["root-a", "root-b"],
  );
  assert.equal(
    tree[0]?.children.map((c) => c.id),
    ["child-a", "child-b"],
  );
});

test("buildCategoryTree treats unknown parent as root", () => {
  const categories = [category("orphan", 1, "missing-parent")];
  const tree = buildCategoryTree(categories);
  assert.equal(tree.map((c) => c.id), ["orphan"]);
});

test("isDescendant finds deep descendants", () => {
  const tree = buildCategoryTree([
    category("root", 1),
    category("mid", 1, "root"),
    category("leaf", 1, "mid"),
  ]);

  assert.ok(isDescendant(tree, "root", "leaf"));
  assert.not.ok(isDescendant(tree, "leaf", "root"));
});

test("isDescendant returns false when ancestor does not exist", () => {
  const tree = buildCategoryTree([category("root", 1), category("leaf", 1, "root")]);
  assert.not.ok(isDescendant(tree, "missing", "leaf"));
});

test("isDescendant does not treat a node as its own descendant", () => {
  const tree = buildCategoryTree([category("root", 1)]);
  assert.not.ok(isDescendant(tree, "root", "root"));
});

test("isLeafCategory only accepts nodes without children", () => {
  const tree = buildCategoryTree([
    category("root", 1),
    category("leaf", 1, "root"),
  ]);

  assert.not.ok(isLeafCategory(tree[0]!));
  assert.ok(isLeafCategory(tree[0]!.children[0]!));
});

test("findCategoryPath returns the full ancestor chain", () => {
  const tree = buildCategoryTree([
    category("vorklinik", 1),
    category("semester-3", 1, "vorklinik"),
    category("ppz", 1, "semester-3"),
    category("klinik", 2),
  ]);

  assert.equal(
    findCategoryPath(tree, "ppz").map((c) => c.id),
    ["vorklinik", "semester-3", "ppz"],
  );
  assert.equal(
    findCategoryPath(tree, "klinik").map((c) => c.id),
    ["klinik"],
  );
});

test("findCategoryPath returns empty for unknown or missing ids", () => {
  const tree = buildCategoryTree([category("root", 1)]);

  assert.equal(findCategoryPath(tree, "nope"), []);
  assert.equal(findCategoryPath(tree, undefined), []);
});

test("findCategoryNode finds nested nodes and tolerates misses", () => {
  const tree = buildCategoryTree([
    category("root", 1),
    category("mid", 1, "root"),
    category("leaf", 1, "mid"),
  ]);

  assert.is(findCategoryNode(tree, "leaf")?.id, "leaf");
  assert.is(findCategoryNode(tree, "missing"), null);
  assert.is(findCategoryNode(tree, undefined), null);
});

test.run();
