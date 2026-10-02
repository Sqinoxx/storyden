import { test } from "uvu";
import * as assert from "uvu/assert";

import type { DatagraphItem } from "@/api/openapi-schema";

import {
  TagItemFilters,
  filterTagItems,
  getTagItemFacets,
  hasActiveTagItemFilters,
} from "./filterTagItems";

function profile(handle: string, name: string) {
  return { handle, name, id: handle, joined: "", roles: [] };
}

function thread(
  id: string,
  opts: {
    title: string;
    createdAt: string;
    category?: { slug: string; name: string };
    author?: ReturnType<typeof profile>;
    replies?: number;
    likes?: number;
    lastReplyAt?: string;
    description?: string;
  },
): DatagraphItem {
  return {
    kind: "thread",
    ref: {
      id,
      title: opts.title,
      slug: id,
      createdAt: opts.createdAt,
      updatedAt: opts.createdAt,
      last_reply_at: opts.lastReplyAt,
      description: opts.description,
      category: opts.category,
      author: opts.author ?? profile("alice", "Alice"),
      reply_status: { replies: opts.replies ?? 0, replied: 0 },
      likes: { likes: opts.likes ?? 0, liked: false },
    },
  } as unknown as DatagraphItem;
}

function node(
  id: string,
  opts: { name: string; createdAt: string; updatedAt?: string },
): DatagraphItem {
  return {
    kind: "node",
    ref: {
      id,
      name: opts.name,
      slug: id,
      description: "",
      createdAt: opts.createdAt,
      updatedAt: opts.updatedAt ?? opts.createdAt,
      owner: profile("bob", "Bob"),
    },
  } as unknown as DatagraphItem;
}

const perio = { slug: "perio", name: "Parodontologie" };
const endo = { slug: "endo", name: "Endodontie" };

const items: DatagraphItem[] = [
  thread("t1", {
    title: "Zahnfleischbluten",
    createdAt: "2026-01-01T00:00:00Z",
    category: perio,
    replies: 5,
    likes: 1,
    lastReplyAt: "2026-06-01T00:00:00Z",
  }),
  thread("t2", {
    title: "Wurzelkanal",
    createdAt: "2026-03-01T00:00:00Z",
    category: endo,
    author: profile("carol", "Carol"),
    replies: 1,
    likes: 9,
    description: "Spülprotokoll gesucht",
  }),
  node("n1", {
    name: "Anatomie",
    createdAt: "2026-02-01T00:00:00Z",
    updatedAt: "2026-07-01T00:00:00Z",
  }),
  { kind: "profile", ref: profile("dave", "Dave") } as unknown as DatagraphItem,
];

const base: TagItemFilters = {
  q: "",
  kind: null,
  category: null,
  author: null,
  sort: "newest",
};

const ids = (f: Partial<TagItemFilters>) =>
  filterTagItems(items, { ...base, ...f }).map((i) => i.ref.id);

test("defaults to newest first and drops unsupported kinds", () => {
  assert.equal(ids({}), ["t2", "n1", "t1"]);
});

test("search matches title, description and author", () => {
  assert.equal(ids({ q: "  WURZEL " }), ["t2"]);
  assert.equal(ids({ q: "spülprotokoll" }), ["t2"]);
  assert.equal(ids({ q: "bob" }), ["n1"]);
  assert.equal(ids({ q: "nothing" }), []);
});

test("filters by kind", () => {
  assert.equal(ids({ kind: "thread" }), ["t2", "t1"]);
  assert.equal(ids({ kind: "node" }), ["n1"]);
});

test("category filter excludes library pages", () => {
  assert.equal(ids({ category: "perio" }), ["t1"]);
});

test("filters by author handle", () => {
  assert.equal(ids({ author: "carol" }), ["t2"]);
  assert.equal(ids({ author: "bob" }), ["n1"]);
});

test("sorts", () => {
  assert.equal(ids({ sort: "oldest" }), ["t1", "n1", "t2"]);
  assert.equal(ids({ sort: "active" }), ["n1", "t1", "t2"]);
  assert.equal(ids({ sort: "replies" }), ["t1", "t2", "n1"]);
  assert.equal(ids({ sort: "likes" }), ["t2", "t1", "n1"]);
  assert.equal(ids({ sort: "alphabetical" }), ["n1", "t2", "t1"]);
});

test("ties fall back to newest", () => {
  assert.equal(
    filterTagItems(
      [
        thread("a", { title: "Same", createdAt: "2026-01-01T00:00:00Z" }),
        thread("b", { title: "same", createdAt: "2026-02-01T00:00:00Z" }),
      ],
      { ...base, sort: "alphabetical" },
    ).map((i) => i.ref.id),
    ["b", "a"],
  );
});

test("facets list unique categories and authors alphabetically", () => {
  const facets = getTagItemFacets(items);
  assert.equal(facets.categories, [
    { value: "endo", label: "Endodontie" },
    { value: "perio", label: "Parodontologie" },
  ]);
  assert.equal(
    facets.authors.map((a) => a.value),
    ["alice", "bob", "carol"],
  );
});

test("detects active filters", () => {
  assert.not.ok(hasActiveTagItemFilters({ ...base, sort: "likes", q: " " }));
  assert.ok(hasActiveTagItemFilters({ ...base, author: "bob" }));
});

test.run();
