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
  enableSemesterGrouping?: boolean;
};

export type ThreadSortOrder = NonNullable<ThreadListParams["sort"]>;

export const SEMESTER_VIEW = "semester";

// Semester grouping happens client side because the term lives in each thread's
// metadata, which the list endpoint cannot sort by. It shares the sort param so
// the menu stays a single mutually exclusive choice.
export type ThreadFeedView = ThreadSortOrder | typeof SEMESTER_VIEW;

const DEFAULT_SORT: ThreadSortOrder = "newest";

// "asc"/"desc" were the pre-API sort values and may still be in shared links.
function parseFeedView(value: string, semesterEnabled: boolean): ThreadFeedView {
  switch (value) {
    case "oldest":
    case "asc":
      return "oldest";
    case "activity":
      return "activity";
    case SEMESTER_VIEW:
      return semesterEnabled ? SEMESTER_VIEW : DEFAULT_SORT;
    default:
      return DEFAULT_SORT;
  }
}

// Grouping maps onto the newest-first query the category route already fetches
// server side, so the SSR fallback stays valid and no refetch flashes on load.
function apiSortFor(view: ThreadFeedView): ThreadSortOrder {
  return view === SEMESTER_VIEW ? DEFAULT_SORT : view;
}

export function useThreadFeedScreen(props: Props) {
  const [page, setPage] = useQueryState("page", {
    ...parseAsInteger,
    defaultValue: props.initialPage ?? 1,
  });
  const [sortParam, setSortParam] = useQueryState(
    "sort",
    parseAsString.withDefault(DEFAULT_SORT),
  );
  const view = parseFeedView(sortParam, props.enableSemesterGrouping ?? false);

  function handlePageChange(page: number) {
    setPage(page);
  }

  const { data, error } = useThreadList(
    {
      page: page.toString(),
      sort: apiSortFor(view),
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

  return {
    ready: true as const,
    data,
    view,
    isGrouped: view === SEMESTER_VIEW,
    handlePageChange,
    handleSetView: (next: ThreadFeedView) =>
      setSortParam(next === DEFAULT_SORT ? null : next),
  };
}
