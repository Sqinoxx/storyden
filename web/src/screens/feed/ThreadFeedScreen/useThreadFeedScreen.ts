"use client";

import { parseAsInteger, parseAsString, useQueryState } from "nuqs";

import { useThreadList } from "@/api/openapi-client/threads";
import {
  Category,
  ThreadListParams,
  ThreadListResult,
} from "@/api/openapi-schema";

export type Props = {
  initialPage?: number;
  initialPageData?: ThreadListResult;
  category:
    | undefined // No category specified, no filters applied.
    | Category // An explicit category.
    | null; // Explicitly uncategorised.
  paginationBasePath: string;
};

export type ThreadSortOrder = NonNullable<ThreadListParams["sort"]>;

const DEFAULT_SORT: ThreadSortOrder = "newest";

// "asc"/"desc" were the pre-API sort values and may still be in shared links.
function parseSortOrder(value: string): ThreadSortOrder {
  switch (value) {
    case "oldest":
    case "asc":
      return "oldest";
    case "activity":
      return "activity";
    default:
      return DEFAULT_SORT;
  }
}

export function useThreadFeedScreen(props: Props) {
  const initialPage = props.initialPage ?? 1;

  const [page, setPage] = useQueryState("page", {
    ...parseAsInteger,
    defaultValue: props.initialPage ?? 1,
  });
  const [sortParam, setSortParam] = useQueryState(
    "sort",
    parseAsString.withDefault(DEFAULT_SORT),
  );
  const sortOrder = parseSortOrder(sortParam);

  function handlePageChange(page: number) {
    setPage(page);
  }

  const { data, error } = useThreadList(
    {
      page: page.toString(),
      sort: sortOrder,
      categories:
        props.category === undefined
          ? []
          : [props.category === null ? "null" : props.category.slug],
    },
    {
      swr: {
        fallbackData: props.initialPageData,
      },
    },
  );
  if (!data) {
    return {
      ready: false as const,
      error,
    };
  }

  const showPaginationTop = data?.next_page && initialPage > 1;

  return {
    ready: true as const,
    showPaginationTop,
    data,
    sortOrder,
    handlePageChange,
    handleSetSortOrder: (order: ThreadSortOrder) =>
      setSortParam(order === DEFAULT_SORT ? null : order),
  };
}
