import type {
  DatagraphItem,
  DatagraphItemNode,
  DatagraphItemThread,
} from "@/api/openapi-schema";

export const TAG_ITEM_SORTS = [
  "newest",
  "oldest",
  "active",
  "replies",
  "likes",
  "alphabetical",
] as const;

export type TagItemSort = (typeof TAG_ITEM_SORTS)[number];

export const TAG_ITEM_KINDS = ["thread", "node"] as const;

export type TagItemKind = (typeof TAG_ITEM_KINDS)[number];

export type TagItem = DatagraphItemThread | DatagraphItemNode;

export type TagItemFilters = {
  q: string;
  kind: TagItemKind | null;
  category: string | null;
  author: string | null;
  sort: TagItemSort;
};

export type TagItemFacet = { value: string; label: string };

export function isTagItem(item: DatagraphItem): item is TagItem {
  return item.kind === "thread" || item.kind === "node";
}

function title(item: TagItem) {
  return item.kind === "thread" ? item.ref.title : item.ref.name;
}

function author(item: TagItem) {
  return item.kind === "thread" ? item.ref.author : item.ref.owner;
}

function time(value: string | undefined) {
  return value ? new Date(value).getTime() : 0;
}

function activity(item: TagItem) {
  return item.kind === "thread"
    ? time(item.ref.last_reply_at ?? item.ref.updatedAt)
    : time(item.ref.updatedAt);
}

function replies(item: TagItem) {
  return item.kind === "thread" ? item.ref.reply_status.replies : 0;
}

function likes(item: TagItem) {
  return item.kind === "thread" ? item.ref.likes.likes : 0;
}

function matchesQuery(item: TagItem, query: string) {
  if (!query) return true;

  const { name, handle } = author(item);
  return [title(item), item.ref.description, name, handle].some((field) =>
    field?.toLowerCase().includes(query),
  );
}

function compare(sort: TagItemSort) {
  const newest = (a: TagItem, b: TagItem) =>
    time(b.ref.createdAt) - time(a.ref.createdAt);

  switch (sort) {
    case "newest":
      return newest;
    case "oldest":
      return (a: TagItem, b: TagItem) =>
        time(a.ref.createdAt) - time(b.ref.createdAt);
    case "active":
      return (a: TagItem, b: TagItem) =>
        activity(b) - activity(a) || newest(a, b);
    case "replies":
      return (a: TagItem, b: TagItem) =>
        replies(b) - replies(a) || newest(a, b);
    case "likes":
      return (a: TagItem, b: TagItem) => likes(b) - likes(a) || newest(a, b);
    case "alphabetical":
      return (a: TagItem, b: TagItem) =>
        title(a).localeCompare(title(b), "de", { sensitivity: "base" }) ||
        newest(a, b);
  }
}

export function filterTagItems(
  items: DatagraphItem[],
  filters: TagItemFilters,
): TagItem[] {
  const query = filters.q.toLowerCase().trim();

  return items
    .filter(isTagItem)
    .filter((item) => !filters.kind || item.kind === filters.kind)
    .filter(
      (item) =>
        !filters.category ||
        (item.kind === "thread" &&
          item.ref.category?.slug === filters.category),
    )
    .filter((item) => !filters.author || author(item).handle === filters.author)
    .filter((item) => matchesQuery(item, query))
    .sort(compare(filters.sort));
}

function byLabel(a: TagItemFacet, b: TagItemFacet) {
  return a.label.localeCompare(b.label, "de", { sensitivity: "base" });
}

export function getTagItemFacets(items: DatagraphItem[]) {
  const categories = new Map<string, string>();
  const authors = new Map<string, string>();

  for (const item of items.filter(isTagItem)) {
    if (item.kind === "thread" && item.ref.category) {
      categories.set(item.ref.category.slug, item.ref.category.name);
    }
    const { handle, name } = author(item);
    authors.set(handle, name);
  }

  const toFacets = (m: Map<string, string>) =>
    Array.from(m, ([value, label]) => ({ value, label })).sort(byLabel);

  return {
    categories: toFacets(categories),
    authors: toFacets(authors),
  };
}

export function hasActiveTagItemFilters(filters: TagItemFilters) {
  return Boolean(
    filters.q.trim() || filters.kind || filters.category || filters.author,
  );
}
