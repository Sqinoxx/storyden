import { z } from "zod";

import { Metadata, ThreadReference } from "@/api/openapi-schema";

// Mirrors app/services/account/semester: summer runs April to September,
// winter runs October to March of the following year. Year is always the year
// the term begins in, so the winter term spanning 2025/26 is { 2025, true }.
export type Term = {
  year: number;
  winter: boolean;
};

export const SEMESTER_META_KEY = "semester";

const SELECTABLE_TERMS_BACK = 16;
const SELECTABLE_TERMS_FORWARD = 1;

export const SemesterMetaSchema = z.object({
  term: z.string(),
});

export function termFor(date: Date | string): Term {
  const d = typeof date === "string" ? new Date(date) : date;
  const year = d.getFullYear();
  const month = d.getMonth() + 1;

  if (month >= 10) {
    return { year, winter: true };
  }

  if (month <= 3) {
    return { year: year - 1, winter: true };
  }

  return { year, winter: false };
}

export function termOrdinal(term: Term): number {
  return term.year * 2 + (term.winter ? 1 : 0);
}

export function termKey(term: Term): string {
  return `${term.year}-${term.winter ? "WS" : "SS"}`;
}

export function parseTermKey(raw: string | undefined): Term | undefined {
  if (!raw) return undefined;

  const separator = raw.indexOf("-");
  if (separator < 0) return undefined;

  const year = Number(raw.slice(0, separator));
  if (!Number.isInteger(year)) return undefined;

  switch (raw.slice(separator + 1).toUpperCase()) {
    case "WS":
      return { year, winter: true };
    case "SS":
      return { year, winter: false };
    default:
      return undefined;
  }
}

function shortYear(year: number): string {
  return String(year % 100).padStart(2, "0");
}

export function formatTermLabel(term: Term): string {
  if (term.winter) {
    return `WS${shortYear(term.year)}/${shortYear(term.year + 1)}`;
  }

  return `SS${shortYear(term.year)}`;
}

export function formatTermKeyLabel(raw: string): string {
  const term = parseTermKey(raw);

  return term ? formatTermLabel(term) : raw;
}

function shiftTerm(term: Term, by: number): Term {
  const ordinal = termOrdinal(term) + by;

  return { year: Math.floor(ordinal / 2), winter: ordinal % 2 === 1 };
}

export function selectableTerms(now: Date): Term[] {
  const current = termFor(now);

  return Array.from(
    { length: SELECTABLE_TERMS_FORWARD + SELECTABLE_TERMS_BACK + 1 },
    (_, i) => shiftTerm(current, SELECTABLE_TERMS_FORWARD - i),
  );
}

export function parseThreadSemesterMeta(
  meta: Metadata | undefined,
): Term | undefined {
  const parsed = SemesterMetaSchema.safeParse(meta?.[SEMESTER_META_KEY]);

  return parsed.success ? parseTermKey(parsed.data.term) : undefined;
}

export function writeThreadSemesterMeta(
  existing: Metadata | undefined,
  term: Term,
): Metadata {
  return {
    ...(existing ?? {}),
    [SEMESTER_META_KEY]: { term: termKey(term) },
  };
}

export function threadTerm(
  thread: Pick<ThreadReference, "meta" | "createdAt">,
): Term {
  return parseThreadSemesterMeta(thread.meta) ?? termFor(thread.createdAt);
}

export type SemesterGroup = {
  key: string;
  term: Term | null;
  threads: ThreadReference[];
};

export const PINNED_GROUP_KEY = "pinned";

// Buckets rather than scanning for runs: a manually backdated thread can sit
// anywhere in the server's date ordering, and a run-based scan would emit a
// second header for a term that already appeared further up the page.
export function groupThreadsBySemester(
  threads: ThreadReference[],
): SemesterGroup[] {
  const pinned = threads.filter((t) => t.pinned);
  const rest = threads.filter((t) => !t.pinned);

  const buckets = new Map<string, SemesterGroup>();

  for (const thread of rest) {
    const term = threadTerm(thread);
    const key = termKey(term);

    const existing = buckets.get(key);
    if (existing) {
      existing.threads.push(thread);
    } else {
      buckets.set(key, { key, term, threads: [thread] });
    }
  }

  const groups = Array.from(buckets.values()).sort(
    (a, b) => termOrdinal(b.term!) - termOrdinal(a.term!),
  );

  if (pinned.length > 0) {
    groups.unshift({ key: PINNED_GROUP_KEY, term: null, threads: pinned });
  }

  return groups;
}
