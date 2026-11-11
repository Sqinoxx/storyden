import {
  Term,
  formatTermLabel,
  termFor,
  termOrdinal,
} from "@/lib/thread/semester";

function termAt(ordinal: number): Term {
  return { year: Math.floor(ordinal / 2), winter: ordinal % 2 === 1 };
}

// Last moment of a term in local time: winter ends 31 March of the following
// year, summer ends 30 September.
export function termEnd(term: Term): Date {
  if (term.winter) {
    return new Date(term.year + 1, 2, 31, 23, 59, 59);
  }
  return new Date(term.year, 8, 30, 23, 59, 59);
}

export type ExpiryPreset = {
  key: "none" | "thisTerm" | "nextTerm";
  term?: Term;
  expiresAt?: Date;
  label?: string;
};

export function expiryPresets(now: Date): ExpiryPreset[] {
  const current = termOrdinal(termFor(now));
  const thisTerm = termAt(current);
  const nextTerm = termAt(current + 1);

  return [
    { key: "none" },
    {
      key: "thisTerm",
      term: thisTerm,
      expiresAt: termEnd(thisTerm),
      label: formatTermLabel(thisTerm),
    },
    {
      key: "nextTerm",
      term: nextTerm,
      expiresAt: termEnd(nextTerm),
      label: formatTermLabel(nextTerm),
    },
  ];
}

// <input type="date"> speaks YYYY-MM-DD in local time; a grant chosen that
// way runs until the end of that day.
export function dateInputValue(d: Date): string {
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function endOfDayFromInput(value: string): Date | undefined {
  const match = /^(\d{4})-(\d{2})-(\d{2})$/.exec(value);
  if (!match) return undefined;
  const [, y, m, d] = match;
  return new Date(Number(y), Number(m) - 1, Number(d), 23, 59, 59);
}
