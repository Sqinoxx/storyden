import { formatDistanceToNow } from "date-fns";
import { de } from "date-fns/locale";

import type { Language } from "@/lib/i18n";

const shortUnits = {
  en: {
    lessThanXSeconds: "{{count}}s",
    xSeconds: "{{count}}s",
    halfAMinute: "30s",
    lessThanXMinutes: "{{count}}min",
    xMinutes: "{{count}}min",
    aboutXHours: "{{count}}h",
    xHours: "{{count}}h",
    xDays: "{{count}}d",
    aboutXWeeks: "{{count}}w",
    xWeeks: "{{count}}w",
    aboutXMonths: "{{count}}mo",
    xMonths: "{{count}}mo",
    aboutXYears: "{{count}}y",
    xYears: "{{count}}y",
    overXYears: "{{count}}y",
    almostXYears: "{{count}}y",
  },
  de: {
    lessThanXSeconds: "{{count}}s",
    xSeconds: "{{count}}s",
    halfAMinute: "30s",
    lessThanXMinutes: "{{count}}min",
    xMinutes: "{{count}}min",
    aboutXHours: "{{count}}h",
    xHours: "{{count}}h",
    xDays: "{{count}}T",
    aboutXWeeks: "{{count}}Wo",
    xWeeks: "{{count}}Wo",
    aboutXMonths: "{{count}}Mon",
    xMonths: "{{count}}Mon",
    aboutXYears: "{{count}}J",
    xYears: "{{count}}J",
    overXYears: "{{count}}J",
    almostXYears: "{{count}}J",
  },
};

type ShortToken = keyof (typeof shortUnits)["en"];

function currentLanguage(): Language {
  if (typeof document === "undefined") return "de";
  return document.documentElement.lang === "en" ? "en" : "de";
}

export const formatDistance = (
  token: ShortToken,
  count: number,
  language: Language = "en",
) => shortUnits[language][token].replace("{{count}}", count.toString());

export function dateFnsLocale(language: Language = currentLanguage()) {
  return language === "de" ? de : undefined;
}

export function timestamp(
  date: string | number | Date,
  short = true,
  language: Language = currentLanguage(),
) {
  try {
    return formatDistanceToNow(
      date,
      short
        ? {
            locale: {
              formatDistance: (token, count) =>
                formatDistance(token, count, language),
            },
          }
        : { addSuffix: true, locale: dateFnsLocale(language) },
    );
  } catch (e: unknown) {
    throw new Error(`Failed to format date: ${date}: error: ${e}`);
  }
}
